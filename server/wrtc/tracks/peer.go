package tracks

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/peer-calls/peer-calls/server/basen"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v2"
)

const (
	rtcpPLIInterval = time.Second * 3
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

type PeerConnection interface {
	AddTrack(*webrtc.Track) (*webrtc.RTPSender, error)
	AddTransceiverFromTrack(track *webrtc.Track, init ...webrtc.RtpTransceiverInit) (*webrtc.RTPTransceiver, error)
	RemoveTrack(*webrtc.RTPSender) error
	OnTrack(func(*webrtc.Track, *webrtc.RTPReceiver))
	WriteRTCP([]rtcp.Packet) error
	NewTrack(uint8, uint32, string, string) (*webrtc.Track, error)
	OnDataChannel(func(*webrtc.DataChannel))
}

type DataTransceiver struct {
	clientID       string
	peerConnection PeerConnection

	mu             sync.RWMutex
	dataChanOnce   sync.Once
	dataChanClosed bool
	dataChannel    *webrtc.DataChannel
	messagesChan   chan webrtc.DataChannelMessage
	closeChannel   chan struct{}
}

func newDataTransceiver(
	clientID string,
	dataChannel *webrtc.DataChannel,
	peerConnection PeerConnection,
) *DataTransceiver {
	d := &DataTransceiver{
		clientID:       clientID,
		peerConnection: peerConnection,
		messagesChan:   make(chan webrtc.DataChannelMessage),
		closeChannel:   make(chan struct{}),
	}
	if dataChannel != nil {
		d.handleDataChannel(dataChannel)
	}
	peerConnection.OnDataChannel(d.handleDataChannel)
	return d
}

func (d *DataTransceiver) handleDataChannel(dataChannel *webrtc.DataChannel) {
	log.Printf("[%s] DataTransceiver.handleDataChannel: %s", d.clientID, dataChannel.Label())
	if dataChannel.Label() == DataChannelName {
		// only want a single data channel for messages and sending files
		d.mu.Lock()
		dataChannel.OnMessage(d.handleMessage)
		d.dataChannel = dataChannel
		d.mu.Unlock()
	}
}

func (d *DataTransceiver) MessagesChannel() <-chan webrtc.DataChannelMessage {
	return d.messagesChan
}

func (d *DataTransceiver) Close() {
	log.Printf("[%s] DataTransceiver.Close", d.clientID)
	d.dataChanOnce.Do(func() {
		close(d.closeChannel)

		d.mu.Lock()
		defer d.mu.Unlock()

		d.dataChanClosed = true
		close(d.messagesChan)
	})
}

func (d *DataTransceiver) handleMessage(msg webrtc.DataChannelMessage) {
	log.Printf("[%s] DataTransceiver.handleMessage", d.clientID)
	d.mu.RLock()
	defer d.mu.RUnlock()

	ch := d.messagesChan
	if d.dataChanClosed {
		ch = nil
	}

	select {
	case ch <- msg:
	case <-d.closeChannel:
	}
}

func (d *DataTransceiver) SendText(message string) (err error) {
	log.Printf("[%s] DataTransceiver.SendText", d.clientID)
	d.mu.RLock()
	if d.dataChannel != nil {
		err = d.dataChannel.SendText(message)
	} else {
		err = fmt.Errorf("[%s] No data channel", d.clientID)
	}
	d.mu.RUnlock()
	return
}

func (d *DataTransceiver) Send(message []byte) (err error) {
	log.Printf("[%s] DataTransceiver.Send", d.clientID)
	d.mu.RLock()
	if d.dataChannel != nil {
		err = d.dataChannel.Send(message)
	} else {
		err = fmt.Errorf("[%s] No data channel", d.clientID)
	}
	d.mu.RUnlock()
	return
}

type peer struct {
	clientID         string
	peerConnection   PeerConnection
	localTracks      []*webrtc.Track
	localTracksMu    sync.RWMutex
	rtpSenderByTrack map[*webrtc.Track]*webrtc.RTPSender

	tracksChannel       chan TrackEvent
	tracksChannelClosed bool
	closeChannel        chan struct{}
	mu                  sync.RWMutex
	closeOnce           sync.Once
}

func newPeer(
	clientID string,
	peerConnection PeerConnection,
) *peer {
	p := &peer{
		clientID:         clientID,
		peerConnection:   peerConnection,
		rtpSenderByTrack: map[*webrtc.Track]*webrtc.RTPSender{},

		tracksChannel: make(chan TrackEvent),
		closeChannel:  make(chan struct{}),
	}

	log.Printf("[%s] Setting PeerConnection.OnTrack listener", clientID)
	peerConnection.OnTrack(p.handleTrack)

	return p
}

// FIXME add support for data channel messages for sending chat messages, and images/files

func (p *peer) Close() {
	p.closeOnce.Do(func() {
		close(p.closeChannel)

		p.mu.Lock()
		defer p.mu.Unlock()

		close(p.tracksChannel)
		p.tracksChannelClosed = true
	})
}

func (p *peer) TracksChannel() <-chan TrackEvent {
	return p.tracksChannel
}

func (p *peer) ClientID() string {
	return p.clientID
}

func (p *peer) AddTrack(track *webrtc.Track) error {
	p.localTracksMu.Lock()
	defer p.localTracksMu.Unlock()

	log.Printf("[%s] peer.AddTrack: add sendonly transceiver for track: %s", p.clientID, track.ID())
	rtpSender, err := p.peerConnection.AddTrack(track)
	// t, err := p.peerConnection.AddTransceiverFromTrack(
	// 	track,
	// 	webrtc.RtpTransceiverInit{
	// 		Direction: webrtc.RTPTransceiverDirectionSendonly,
	// 	},
	// )

	if err != nil {
		return fmt.Errorf("[%s] peer.AddTrack: error adding track: %s: %s", p.clientID, track.ID(), err)
	}

	// p.rtpSenderByTrack[track] = t.Sender()
	p.rtpSenderByTrack[track] = rtpSender
	return nil
}

func (p *peer) RemoveTrack(track *webrtc.Track) error {
	p.localTracksMu.Lock()
	defer p.localTracksMu.Unlock()
	log.Printf("[%s] peer.RemoveTrack: %s", p.clientID, track.ID())
	rtpSender, ok := p.rtpSenderByTrack[track]
	if !ok {
		return fmt.Errorf("[%s] peer.RemoveTrack: cannot find sender for track: %s", p.clientID, track.ID())
	}
	delete(p.rtpSenderByTrack, track)
	return p.peerConnection.RemoveTrack(rtpSender)
}

func (p *peer) handleTrack(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
	log.Printf("[%s] peer.handleTrack (id: %s, label: %s, type: %s, ssrc: %d)",
		p.clientID, remoteTrack.ID(), remoteTrack.Label(), remoteTrack.Kind(), remoteTrack.SSRC())
	localTrack, err := p.startCopyingTrack(remoteTrack)
	if err != nil {
		log.Printf("Error copying remote track: %s", err)
		return
	}
	p.localTracksMu.Lock()
	p.localTracks = append(p.localTracks, localTrack)
	p.localTracksMu.Unlock()

	log.Printf("[%s] peer.handleTrack add track to list of local tracks: %s", p.clientID, localTrack.ID())
	p.tracksChannel <- TrackEvent{p.clientID, localTrack, TrackEventTypeAdd}
}

func (p *peer) sendTrackEvent(t TrackEvent) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ch := p.tracksChannel
	if p.tracksChannelClosed {
		ch = nil
	}

	select {
	case ch <- t:
		log.Printf("[%s] sendTrackEvent success", p.clientID)
	case <-p.closeChannel:
		log.Printf("[%s] sendTrackEvent channel closed", p.clientID)
	}
}

func (p *peer) Tracks() []*webrtc.Track {
	return p.localTracks
}

func (p *peer) startCopyingTrack(remoteTrack *webrtc.Track) (*webrtc.Track, error) {
	remoteTrackID := remoteTrack.ID()
	if remoteTrackID == "" {
		remoteTrackID = basen.NewUUIDBase62()
	}
	// this is the media stream ID we add the p.clientID in the string to know
	// which user the video came from and the remoteTrack.Label() so we can
	// associate audio/video tracks from the same MediaStream
	remoteTrackLabel := remoteTrack.Label()
	if remoteTrackLabel == "" {
		remoteTrackLabel = basen.NewUUIDBase62()
	}
	localTrackLabel := "sfu_" + p.clientID + "_" + remoteTrackLabel

	localTrackID := "sfu_" + remoteTrackID
	log.Printf("[%s] peer.startCopyingTrack: (id: %s, label: %s) to (id: %s, label: %s), ssrc: %d",
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

	ticker := time.NewTicker(rtcpPLIInterval)
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
				log.Printf("[%s] Error sending rtcp PLI for local track: %s: %s",
					p.clientID,
					localTrackID,
					err,
				)
			}
		}

		writeRTCP()
		for range ticker.C {
			writeRTCP()
		}
	}()

	go func() {
		defer ticker.Stop()
		defer func() {
			p.mu.RLock()
			if !p.tracksChannelClosed {
				p.tracksChannel <- TrackEvent{p.clientID, localTrack, TrackEventTypeRemove}
			}
			p.mu.RUnlock()
		}()
		rtpBuf := make([]byte, 1400)
		for {
			i, err := remoteTrack.Read(rtpBuf)
			if err != nil {
				log.Printf(
					"[%s] Error reading from remote track: %s: %s",
					p.clientID,
					remoteTrack.ID(),
					err,
				)
				return
			}

			// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
			if _, err = localTrack.Write(rtpBuf[:i]); err != nil && err != io.ErrClosedPipe {
				log.Printf(
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
