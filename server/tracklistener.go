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
}

type TrackInfo struct {
	RTPTransceiver *webrtc.RTPTransceiver
	RTPSender      *webrtc.RTPSender
	TrackMetadata  TrackMetadata
}

type trackListener struct {
	log              Logger
	clientID         string
	peerConnection   *webrtc.PeerConnection
	localTracks      []*webrtc.Track
	localTracksMu    sync.RWMutex
	trackInfoByTrack map[*webrtc.Track]TrackInfo
	onTrackEvent     func(TrackEvent)
	mu               sync.RWMutex
	pliInterval      time.Duration
	ssrcMaxBitrates  map[uint32]uint64
}

func newTrackListener(
	loggerFactory LoggerFactory,
	clientID string,
	peerConnection *webrtc.PeerConnection,
	onTrackEvent func(TrackEvent),
) *trackListener {
	p := &trackListener{
		log:              loggerFactory.GetLogger("peer"),
		clientID:         clientID,
		peerConnection:   peerConnection,
		trackInfoByTrack: map[*webrtc.Track]TrackInfo{},
		onTrackEvent:     onTrackEvent,
		ssrcMaxBitrates:  map[uint32]uint64{},
	}

	p.log.Printf("[%s] Setting PeerConnection.OnTrack listener", clientID)
	peerConnection.OnTrack(p.handleTrack)

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

	for _, trackInfo := range p.trackInfoByTrack {
		m := trackInfo.TrackMetadata
		m.Mid = trackInfo.RTPTransceiver.Mid()
		metadata = append(metadata, m)
	}
	return
}

func (p *trackListener) WriteRTCP(anyPacket rtcp.Packet) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch pkt := anyPacket.(type) {
	case *rtcp.PictureLossIndication:
		return p.peerConnection.WriteRTCP([]rtcp.Packet{pkt})
	case *rtcp.SourceDescription:
	case *rtcp.ReceiverEstimatedMaximumBitrate:
		bitrate, ok := p.ssrcMaxBitrates[pkt.SenderSSRC]
		if !ok || pkt.Bitrate < bitrate {
			p.ssrcMaxBitrates[pkt.SenderSSRC] = bitrate
			return p.peerConnection.WriteRTCP([]rtcp.Packet{pkt})
		}
	case *rtcp.ReceiverReport:
	case *rtcp.SenderReport:
	default:
		p.log.Printf("[%s] Got unhandled RTCP pkt for track: %d (%T)", p.clientID, pkt.DestinationSSRC(), pkt)
	}

	return nil
}

func (p *trackListener) AddTrack(sourceClientID string, track *webrtc.Track) (chan rtcp.Packet, error) {
	p.localTracksMu.Lock()
	defer p.localTracksMu.Unlock()

	p.log.Printf("[%s] peer.AddTrack: add sendonly transceiver for track: %s", p.clientID, track.ID())
	rtpSender, err := p.peerConnection.AddTrack(track)

	var transceiver *webrtc.RTPTransceiver
	for _, tr := range p.peerConnection.GetTransceivers() {
		if tr.Sender() == rtpSender {
			transceiver = tr
			break
		}
	}

	rtcpCh := make(chan rtcp.Packet)

	if err != nil {
		close(rtcpCh)
		return rtcpCh, fmt.Errorf("[%s] peer.AddTrack: error adding track: %s: %s", p.clientID, track.ID(), err)
	}

	go func() {
		for {
			rtcps, err := rtpSender.ReadRTCP()
			if err != nil {
				p.log.Printf("[%s] Error reading rtcp for sender track: %d: %s",
					p.clientID,
					track.SSRC(),
					err,
				)
				close(rtcpCh)
				break
			}
			for _, pkt := range rtcps {
				rtcpCh <- pkt
			}
		}
	}()

	p.trackInfoByTrack[track] = TrackInfo{
		RTPSender:      rtpSender,
		RTPTransceiver: transceiver,
		TrackMetadata: TrackMetadata{
			Mid:      "",
			Kind:     track.Kind().String(),
			UserID:   sourceClientID,
			StreamID: track.Label(),
		},
	}
	return rtcpCh, nil
}

func (p *trackListener) RemoveTrack(track *webrtc.Track) error {
	p.localTracksMu.Lock()
	defer p.localTracksMu.Unlock()
	p.log.Printf("[%s] peer.RemoveTrack: %s", p.clientID, track.ID())
	trackInfo, ok := p.trackInfoByTrack[track]
	if !ok {
		return fmt.Errorf("[%s] peer.RemoveTrack: cannot find sender for track: %s", p.clientID, track.ID())
	}
	delete(p.trackInfoByTrack, track)
	delete(p.ssrcMaxBitrates, track.SSRC())
	return p.peerConnection.RemoveTrack(trackInfo.RTPSender)
}

func (p *trackListener) handleTrack(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
	p.log.Printf("[%s] peer.handleTrack (id: %s, label: %s, type: %s, ssrc: %d)",
		p.clientID, remoteTrack.ID(), remoteTrack.Label(), remoteTrack.Kind(), remoteTrack.SSRC())
	localTrack, err := p.startCopyingTrack(remoteTrack, receiver)
	if err != nil {
		p.log.Printf("Error copying remote track: %s", err)
		return
	}
	p.localTracksMu.Lock()
	p.localTracks = append(p.localTracks, localTrack)
	p.localTracksMu.Unlock()

	p.log.Printf("[%s] peer.handleTrack add track to list of local tracks: %s", p.clientID, localTrack.ID())

	p.sendTrackEvent(TrackEvent{p.clientID, localTrack, TrackEventTypeAdd})
}

func (p *trackListener) sendTrackEvent(t TrackEvent) {
	go p.onTrackEvent(t)
}

func (p *trackListener) Tracks() []*webrtc.Track {
	return p.localTracks
}

func (p *trackListener) startCopyingTrack(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) (*webrtc.Track, error) {
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
	p.log.Printf("[%s] peer.startCopyingTrack: (id: %s, label: %s) to (id: %s, label: %s), ssrc: %d",
		p.clientID, remoteTrack.ID(), remoteTrack.Label(), localTrackID, localTrackLabel, remoteTrack.SSRC())

	ssrc := remoteTrack.SSRC()
	// Create a local track, all our SFU clients will be fed via this track
	localTrack, err := p.peerConnection.NewTrack(remoteTrack.PayloadType(), ssrc, localTrackID, localTrackLabel)
	if err != nil {
		err = fmt.Errorf("[%s] peer.startCopyingTrack: error creating new track, trackID: %s, error: %s", p.clientID, remoteTrack.ID(), err)
		return nil, err
	}

	// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
	// This can be less wasteful by processing incoming RTCP events, then we would emit a NACK/PLI when a viewer requests it

	var ticker *time.Ticker
	if p.pliInterval > 0 {
		ticker = time.NewTicker(p.pliInterval)
		go func() {
			writeRTCP := func() {
				err := p.peerConnection.WriteRTCP(
					[]rtcp.Packet{
						&rtcp.PictureLossIndication{
							MediaSSRC: ssrc,
						},
					},
				)
				if err != nil {
					p.log.Printf("[%s] Error sending rtcp PLI for local track: %s: %s",
						p.clientID,
						localTrackID,
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
		defer p.sendTrackEvent(TrackEvent{p.clientID, localTrack, TrackEventTypeRemove})
		if ticker != nil {
			defer ticker.Stop()
		}
		for {
			pkt, err := remoteTrack.ReadRTP()
			if err != nil {
				p.log.Printf(
					"[%s] Error reading from remote track: %s: %s",
					p.clientID,
					remoteTrack.ID(),
					err,
				)
				return
			}

			// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
			err = localTrack.WriteRTP(pkt)
			if err != nil && err != io.ErrClosedPipe {
				p.log.Printf(
					"[%s] Error writing to local track: %s: %s",
					p.clientID,
					localTrackID,
					err,
				)
				return
			}
		}
	}()

	return localTrack, nil
}
