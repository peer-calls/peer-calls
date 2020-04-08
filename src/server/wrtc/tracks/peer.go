package tracks

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/jeremija/peer-calls/src/server/basen"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v2"
)

const (
	rtcpPLIInterval = time.Second * 3
)

type PeerConnection interface {
	AddTrack(*webrtc.Track) (*webrtc.RTPSender, error)
	AddTransceiverFromTrack(track *webrtc.Track, init ...webrtc.RtpTransceiverInit) (*webrtc.RTPTransceiver, error)
	RemoveTrack(*webrtc.RTPSender) error
	OnTrack(func(*webrtc.Track, *webrtc.RTPReceiver))
	OnICEConnectionStateChange(func(webrtc.ICEConnectionState))
	WriteRTCP([]rtcp.Packet) error
	NewTrack(uint8, uint32, string, string) (*webrtc.Track, error)
}

type peer struct {
	clientID         string
	peerConnection   PeerConnection
	localTracks      []*webrtc.Track
	localTracksMu    sync.RWMutex
	rtpSenderByTrack map[*webrtc.Track]*webrtc.RTPSender
	onTrack          func(clientID string, track *webrtc.Track)
	onClose          func(clientID string)
}

func newPeer(
	clientID string,
	peerConnection PeerConnection,
	onTrack func(clientID string, track *webrtc.Track),
	onClose func(clientID string),
) *peer {
	p := &peer{
		clientID:         clientID,
		peerConnection:   peerConnection,
		onTrack:          onTrack,
		onClose:          onClose,
		rtpSenderByTrack: map[*webrtc.Track]*webrtc.RTPSender{},
	}

	peerConnection.OnICEConnectionStateChange(p.handleICEConnectionStateChange)
	log.Printf("[%s] Setting PeerConnection.OnTrack listener", clientID)
	peerConnection.OnTrack(p.handleTrack)

	return p
}

// FIXME add support for data channel messages for sending chat messages, and images/files

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

func (p *peer) handleICEConnectionStateChange(connectionState webrtc.ICEConnectionState) {
	log.Printf("[%s] Peer connection state changed: %s", p.clientID, connectionState.String())
	// if connectionState == webrtc.ICEConnectionStateClosed ||
	// 	connectionState == webrtc.ICEConnectionStateDisconnected ||
	// 	connectionState == webrtc.ICEConnectionStateFailed {
	// }

	if connectionState == webrtc.ICEConnectionStateDisconnected {
		p.onClose(p.clientID)
	}
}

func (p *peer) handleTrack(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
	log.Printf("[%s] peer.handleTrack (id: %s, label: %s, type: %s)",
		p.clientID, remoteTrack.ID(), remoteTrack.Label(), remoteTrack.Kind())
	localTrack, err := p.startCopyingTrack(remoteTrack)
	if err != nil {
		log.Printf("Error copying remote track: %s", err)
		return
	}
	p.localTracksMu.Lock()
	p.localTracks = append(p.localTracks, localTrack)
	p.localTracksMu.Unlock()

	log.Printf("[%s] peer.handleTrack add track to list of local tracks: %s", p.clientID, localTrack.ID())
	p.onTrack(p.clientID, localTrack)
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
	localTrackLabel := "sfu__" + p.clientID + "__" + remoteTrackLabel

	localTrackID := "sfu__" + remoteTrackID
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
		for range ticker.C {
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
	}()

	go func() {
		defer ticker.Stop()
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
