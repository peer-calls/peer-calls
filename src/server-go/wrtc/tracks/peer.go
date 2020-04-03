package tracks

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/jeremija/peer-calls/src/server-go/basen"
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
	log.Printf("Adding track listener for clientID: %s", clientID)
	peerConnection.OnTrack(p.handleTrack)

	return p
}

// FIXME add support for data channel messages for sending chat messages, and images/files

func (p *peer) ClientID() string {
	return p.clientID
}

func (p *peer) AddTrack(track *webrtc.Track) error {
	log.Printf("Add track: %s to peer clientID: %s", track.ID(), p.clientID)

	// rtpSender, err := p.peerConnection.AddTrack(track)
	t, err := p.peerConnection.AddTransceiverFromTrack(
		track,
		webrtc.RtpTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionSendonly,
		},
	)

	if err != nil {
		return fmt.Errorf("Peer.AddTrack Error adding track: %s to peer clientID: %s: %s", track.ID(), p.clientID, err)
	}

	p.rtpSenderByTrack[track] = t.Sender()
	// p.rtpSenderByTrack[track] = rtpSender
	return nil
}

func (p *peer) RemoveTrack(track *webrtc.Track) error {
	log.Printf("Remove track: %s from peer clientID: %s", track.ID(), p.clientID)
	rtpSender, ok := p.rtpSenderByTrack[track]
	if !ok {
		return fmt.Errorf("Cannot find sender for track: %s, clientID: %s", track.ID(), p.clientID)
	}
	return p.peerConnection.RemoveTrack(rtpSender)
}

func (p *peer) handleICEConnectionStateChange(connectionState webrtc.ICEConnectionState) {
	log.Printf("Peer connection state changed, clientID: %s, state: %s",
		p.clientID,
		connectionState.String(),
	)
	// if connectionState == webrtc.ICEConnectionStateClosed ||
	// 	connectionState == webrtc.ICEConnectionStateDisconnected ||
	// 	connectionState == webrtc.ICEConnectionStateFailed {
	// }

	if connectionState == webrtc.ICEConnectionStateDisconnected {
		p.onClose(p.clientID)
	}
}

func (p *peer) handleTrack(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
	log.Printf("handleTrack %s for clientID: %s (type: %s)", remoteTrack.ID(), p.clientID, remoteTrack.Kind())
	localTrack, err := p.startCopyingTrack(remoteTrack)
	if err != nil {
		log.Printf("Error copying remote track: %s", err)
		return
	}
	p.localTracksMu.Lock()
	p.localTracks = append(p.localTracks, localTrack)
	p.localTracksMu.Unlock()

	log.Printf("handleTrack add track to list of local tracks: %s for clientID: %s", localTrack.ID(), p.clientID)
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
	remoteTrackLabel := remoteTrack.Label()
	if remoteTrackLabel == "" {
		remoteTrackLabel = remoteTrackID
	}
	localTrackID := "copy-" + p.clientID + "-" + remoteTrackID
	log.Printf("startCopyingTrack: %s to %s for peer clientID: %s, ssrc: %d", remoteTrack.ID(), localTrackID, p.clientID, remoteTrack.SSRC())

	// Create a local track, all our SFU clients will be fed via this track
	localTrack, err := p.peerConnection.NewTrack(remoteTrack.PayloadType(), remoteTrack.SSRC(), localTrackID, remoteTrackLabel)
	if err != nil {
		err = fmt.Errorf("startCopyingTrack: error creating new track, trackID: %s, clientID: %s, error: %s", remoteTrack.ID(), p.clientID, err)
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
						MediaSSRC: remoteTrack.SSRC(),
					},
				},
			)
			if err != nil {
				log.Printf("Error sending rtcp PLI for local track: %s for clientID: %s: %s",
					localTrackID,
					p.clientID,
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
					"Error reading from remote track: %s for clientID: %s: %s",
					remoteTrack.ID(),
					p.clientID,
					err,
				)
				return
			}

			// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
			if _, err = localTrack.Write(rtpBuf[:i]); err != nil && err != io.ErrClosedPipe {
				log.Printf(
					"Error writing to local track: %s for clientID: %s: %s",
					localTrackID,
					p.clientID,
					err,
				)
				return
			}
		}
	}()

	return localTrack, nil
}
