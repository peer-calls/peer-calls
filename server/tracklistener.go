package server

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v2"
)

type TrackEventType uint32

const (
	TrackEventTypeAdd = iota + 1
	TrackEventTypeRemove
)

type TrackEvent struct {
	ClientID string
	Track    *webrtc.Track
	Type     TrackEventType
}

type TrackMetadata struct {
	Mid      string `json:"mid"`
	UserID   string `json:"userId"`
	StreamID string `json:"streamId"`
	Kind     string `json:"kind"`
	SSRC     uint32
}

type trackListener struct {
	log           Logger
	clientID      string
	transport     Transport
	localTracks   []*webrtc.Track
	localTracksMu sync.RWMutex
	trackMetadata map[*webrtc.Track]TrackMetadata
	onTrackEvent  func(TrackEvent)
	mu            sync.RWMutex
	pliInterval   time.Duration
	jitterHandler JitterHandler
}

func newTrackListener(
	loggerFactory LoggerFactory,
	clientID string,
	transport Transport,
	onTrackEvent func(TrackEvent),
	nackHandler JitterHandler,
) *trackListener {
	p := &trackListener{
		log:           loggerFactory.GetLogger("tracklistener"),
		clientID:      clientID,
		transport:     transport,
		trackMetadata: map[*webrtc.Track]TrackMetadata{},
		onTrackEvent:  onTrackEvent,
		jitterHandler: nackHandler,
	}

	p.log.Printf("[%s] Setting PeerConnection.OnTrack listener", clientID)
	transport.OnTrack(p.handleTrack)

	return p
}

func (p *trackListener) ClientID() string {
	return p.clientID
}

// GetTracksMetadata gets metadata of the sending tracks with updated Mid
func (p *trackListener) GetTracksMetadata() (metadata []TrackMetadata) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	metadata = make([]TrackMetadata, 0)

	for _, d := range p.trackMetadata {
		d.Mid = p.transport.Mid(d.SSRC)
		metadata = append(metadata, d)
	}
	return
}

func (p *trackListener) WriteRTCP(anyPacket rtcp.Packet) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch pkt := anyPacket.(type) {
	case *rtcp.PictureLossIndication, *rtcp.TransportLayerNack, *rtcp.ReceiverEstimatedMaximumBitrate:
		prometheusRTCPPacketsSent.Inc()
		return p.transport.WriteRTCP([]rtcp.Packet{pkt})
	case *rtcp.SourceDescription:
	case *rtcp.ReceiverReport:
	case *rtcp.SenderReport:
	default:
		p.log.Printf("[%s] Got unhandled RTCP pkt for track: %d (%T)", p.clientID, pkt.DestinationSSRC(), pkt)
	}

	return nil
}

func (p *trackListener) AddTrack(sourceClientID string, track *webrtc.Track) (<-chan rtcp.Packet, error) {
	p.localTracksMu.Lock()
	defer p.localTracksMu.Unlock()

	p.log.Printf("[%s] peer.AddTrack: %d", p.clientID, track.SSRC())
	rtcpChSource, err := p.transport.AddTrack(track)

	if err != nil {
		return rtcpChSource, fmt.Errorf("[%s] peer.AddTrack: error adding track: %d: %s", p.clientID, track.SSRC(), err)
	}

	rtcpCh := make(chan rtcp.Packet)

	go func() {
		defer close(rtcpCh)
		for pkt := range rtcpChSource {
			prometheusRTCPPacketsReceived.Inc()
			switch rtcpPkt := pkt.(type) {
			case *rtcp.TransportLayerNack:
				nack := p.jitterHandler.HandleNack(p.clientID, p.transport, rtcpPkt)
				if nack != nil {
					rtcpCh <- nack
				}
			default:
				rtcpCh <- pkt
			}
		}
	}()

	p.trackMetadata[track] = TrackMetadata{
		Mid:      "",
		Kind:     track.Kind().String(),
		UserID:   sourceClientID,
		StreamID: track.Label(),
		SSRC:     track.SSRC(),
	}
	return rtcpCh, nil
}

func (p *trackListener) RemoveTrack(track *webrtc.Track) error {
	p.localTracksMu.Lock()
	defer p.localTracksMu.Unlock()
	p.log.Printf("[%s] peer.RemoveTrack: %d", p.clientID, track.SSRC())
	delete(p.trackMetadata, track)
	return p.transport.RemoveTrack(track)
}

func (p *trackListener) handleTrack(remoteTrack *webrtc.Track) {
	p.log.Printf("[%s] peer.handleTrack (id: %s, label: %s, type: %s, ssrc: %d)",
		p.clientID, remoteTrack.ID(), remoteTrack.Label(), remoteTrack.Kind(), remoteTrack.SSRC())
	localTrack, err := p.startCopyingTrack(remoteTrack)
	if err != nil {
		p.log.Printf("Error copying remote track: %s", err)
		return
	}
	p.localTracksMu.Lock()
	p.localTracks = append(p.localTracks, localTrack)
	p.localTracksMu.Unlock()

	p.log.Printf("[%s] peer.handleTrack add track to list of local tracks: %d", p.clientID, localTrack.SSRC())

	p.sendTrackEvent(TrackEvent{p.clientID, localTrack, TrackEventTypeAdd})
}

func (p *trackListener) sendTrackEvent(t TrackEvent) {
	go p.onTrackEvent(t)
}

func (p *trackListener) Tracks() []*webrtc.Track {
	return p.localTracks
}

func (p *trackListener) startCopyingTrack(remoteTrack *webrtc.Track) (*webrtc.Track, error) {
	remoteTrackID := remoteTrack.ID()
	if remoteTrackID == "" {
		remoteTrackID = NewUUIDBase62()
	}
	// this is the media stream ID we add the p.clientID in the string to know
	// which user the video came from and the remoteTrack.Label() so we can
	// associate audio/video tracks from the same MediaStream
	remoteTrackLabel := remoteTrack.Label()
	if remoteTrackLabel == "" {
		remoteTrackLabel = NewUUIDBase62()
	}
	localTrackLabel := "sfu_" + p.clientID + "_" + remoteTrackLabel

	localTrackID := "sfu_" + remoteTrackID
	p.log.Printf("[%s] peer.startCopyingTrack: %d", p.clientID, remoteTrack.SSRC())

	ssrc := remoteTrack.SSRC()
	// Create a local track, all our SFU clients will be fed via this track
	localTrack, err := p.transport.NewTrack(remoteTrack.PayloadType(), ssrc, localTrackID, localTrackLabel)
	if err != nil {
		err = fmt.Errorf("[%s] peer.startCopyingTrack: error creating new track: %d, error: %s", p.clientID, remoteTrack.SSRC(), err)
		return nil, err
	}

	// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
	// This can be less wasteful by processing incoming RTCP events, then we would emit a NACK/PLI when a viewer requests it

	var ticker *time.Ticker
	if p.pliInterval > 0 {
		ticker = time.NewTicker(p.pliInterval)
		go func() {
			writeRTCP := func() {
				err := p.transport.WriteRTCP(
					[]rtcp.Packet{
						&rtcp.PictureLossIndication{
							MediaSSRC: ssrc,
						},
					},
				)
				if err != nil {
					p.log.Printf("[%s] Error sending rtcp PLI for local track: %d: %s",
						p.clientID,
						localTrack.SSRC(),
						err,
					)
				}
			}
			for range ticker.C {
				writeRTCP()
			}
		}()
	}

	go func() {
		start := time.Now()
		prometheusWebRTCTracksTotal.Inc()
		prometheusWebRTCTracksActive.Inc()
		defer func() {
			prometheusWebRTCTracksActive.Dec()
			prometheusWebRTCTracksDuration.Observe(time.Now().Sub(start).Seconds())
		}()
		defer p.sendTrackEvent(TrackEvent{p.clientID, localTrack, TrackEventTypeRemove})
		if ticker != nil {
			defer ticker.Stop()
		}
		for {
			pkt, err := remoteTrack.ReadRTP()
			if err != nil {
				p.log.Printf("[%s] RTP stream for track: %d has ended: %s", p.clientID, remoteTrack.SSRC(), err)
				return
			}

			prometheusRTPPacketsReceived.Inc()
			p.jitterHandler.HandleRTP(p.clientID, p.transport, pkt)

			// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
			err = localTrack.WriteRTP(pkt)
			if err != nil && err != io.ErrClosedPipe {
				p.log.Printf(
					"[%s] Error writing to local track: %d: %s",
					p.clientID,
					localTrack.SSRC(),
					err,
				)
				p.jitterHandler.RemoveBuffer(remoteTrack.SSRC())
				return
			}
		}
	}()

	return localTrack, nil
}
