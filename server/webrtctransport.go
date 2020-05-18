package server

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

type TrackInfo struct {
	PayloadType uint8
	SSRC        uint32
	ID          string
	Label       string
	Kind        webrtc.RTPCodecType
	Mid         string
}

type TrackEventType uint8

const (
	TrackEventTypeAdd TrackEventType = iota + 1
	TrackEventTypeRemove
)

type TrackEvent struct {
	TrackInfo
	Type TrackEventType
}

type WebRTCTransportFactory struct {
	loggerFactory LoggerFactory
	iceServers    []ICEServer
	webrtcAPI     *webrtc.API
}

func NewWebRTCTransportFactory(
	loggerFactory LoggerFactory,
	iceServers []ICEServer,
	sfuConfig NetworkConfigSFU,
) *WebRTCTransportFactory {

	allowedInterfaces := map[string]struct{}{}
	for _, iface := range sfuConfig.Interfaces {
		allowedInterfaces[iface] = struct{}{}
	}

	settingEngine := webrtc.SettingEngine{
		LoggerFactory: NewPionLoggerFactory(loggerFactory),
	}
	if len(allowedInterfaces) > 0 {
		settingEngine.SetInterfaceFilter(func(iface string) bool {
			_, ok := allowedInterfaces[iface]
			return ok
		})
	}
	settingEngine.SetTrickle(true)
	var mediaEngine webrtc.MediaEngine
	RegisterCodecs(&mediaEngine, sfuConfig.JitterBuffer)
	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(settingEngine),
	)

	return &WebRTCTransportFactory{loggerFactory, iceServers, api}
}

func RegisterCodecs(mediaEngine *webrtc.MediaEngine, jitterBufferEnabled bool) {
	mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	rtcpfb := []webrtc.RTCPFeedback{
		webrtc.RTCPFeedback{
			Type: webrtc.TypeRTCPFBGoogREMB,
		},
		// webrtc.RTCPFeedback{
		// 	Type:      webrtc.TypeRTCPFBCCM,
		// 	Parameter: "fir",
		// },

		// https://tools.ietf.org/html/rfc4585#section-4.2
		// "pli" indicates the use of Picture Loss Indication feedback as defined
		// in Section 6.3.1.
		webrtc.RTCPFeedback{
			Type:      webrtc.TypeRTCPFBNACK,
			Parameter: "pli",
		},
	}

	if jitterBufferEnabled {
		// The feedback type "nack", without parameters, indicates use of the
		// Generic NACK feedback format as defined in Section 6.2.1.
		rtcpfb = append(rtcpfb, webrtc.RTCPFeedback{
			Type: webrtc.TypeRTCPFBNACK,
		})
	}

	mediaEngine.RegisterCodec(webrtc.NewRTPVP8CodecExt(webrtc.DefaultPayloadTypeVP8, 90000, rtcpfb, ""))
	// s.mediaEngine.RegisterCodec(webrtc.NewRTPH264CodecExt(webrtc.DefaultPayloadTypeH264, 90000, rtcpfb, IOSH264Fmtp))
	// s.mediaEngine.RegisterCodec(webrtc.NewRTPVP9Codec(webrtc.DefaultPayloadTypeVP9, 90000))
}

type WebRTCTransport struct {
	mu sync.Mutex
	wg sync.WaitGroup

	log     Logger
	rtpLog  Logger
	rtcpLog Logger

	clientID        string
	peerConnection  *webrtc.PeerConnection
	signaller       *Signaller
	dataTransceiver *DataTransceiver

	trackEventsCh chan TrackEvent
	rtpCh         chan *rtp.Packet
	rtcpCh        chan rtcp.Packet

	localTracks  map[uint32]localTrackInfo
	remoteTracks map[uint32]remoteTrackInfo
}

var _ Transport = &WebRTCTransport{}

func (f WebRTCTransportFactory) NewWebRTCTransport(clientID string) (*WebRTCTransport, error) {
	webrtcICEServers := []webrtc.ICEServer{}
	for _, iceServer := range GetICEAuthServers(f.iceServers) {
		var c webrtc.ICECredentialType
		if iceServer.Username != "" && iceServer.Credential != "" {
			c = webrtc.ICECredentialTypePassword
		}
		webrtcICEServers = append(webrtcICEServers, webrtc.ICEServer{
			URLs:           iceServer.URLs,
			CredentialType: c,
			Username:       iceServer.Username,
			Credential:     iceServer.Credential,
		})
	}

	webrtcConfig := webrtc.Configuration{
		ICEServers: webrtcICEServers,
	}

	peerConnection, err := f.webrtcAPI.NewPeerConnection(webrtcConfig)
	if err != nil {
		return nil, err
	}

	return NewWebRTCTransport(f.loggerFactory, clientID, true, peerConnection)
}

func NewWebRTCTransport(loggerFactory LoggerFactory, clientID string, initiator bool, peerConnection *webrtc.PeerConnection) (*WebRTCTransport, error) {
	signaller, err := NewSignaller(
		loggerFactory,
		initiator,
		peerConnection,
		localPeerID,
		clientID,
	)

	log := loggerFactory.GetLogger("webrtctransport")

	peerConnection.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		log.Printf("[%s] ICE gathering state changed: %s", clientID, state)
	})

	closePeer := func(reason error) error {
		err = peerConnection.Close()
		if err != nil {
			return fmt.Errorf("Error closing peer connection: %s. Close was called because: %w", err, reason)
		} else {
			return reason
		}
	}

	var dataChannel *webrtc.DataChannel
	if initiator {
		// need to do this to connect with simple peer
		// only when we are the initiator
		dataChannel, err = peerConnection.CreateDataChannel("data", nil)
		if err != nil {
			return nil, closePeer(fmt.Errorf("Error creating data channel: %w", err))
		}
	}
	dataTransceiver := NewDataTransceiver(loggerFactory, clientID, dataChannel, peerConnection)

	if err != nil {
		return nil, closePeer(fmt.Errorf("Error initializing signaller: %w", err))
	}

	rtpLog := loggerFactory.GetLogger("rtp")
	rtcpLog := loggerFactory.GetLogger("rtcp")

	transport := &WebRTCTransport{
		log:     log,
		rtpLog:  rtpLog,
		rtcpLog: rtcpLog,

		clientID:        clientID,
		signaller:       signaller,
		peerConnection:  peerConnection,
		dataTransceiver: dataTransceiver,

		trackEventsCh: make(chan TrackEvent),
		rtpCh:         make(chan *rtp.Packet),
		rtcpCh:        make(chan rtcp.Packet),

		localTracks:  map[uint32]localTrackInfo{},
		remoteTracks: map[uint32]remoteTrackInfo{},
	}
	peerConnection.OnTrack(transport.handleTrack)

	go func() {
		// wait for peer connection to be closed
		<-signaller.CloseChannel()
		// do not close channels before all writing goroutines exit
		transport.wg.Wait()
		transport.dataTransceiver.Close()
		close(transport.rtpCh)
		close(transport.rtcpCh)
		close(transport.trackEventsCh)
	}()
	return transport, nil
}

type localTrackInfo struct {
	trackInfo   TrackInfo
	transceiver *webrtc.RTPTransceiver
	sender      *webrtc.RTPSender
	track       *webrtc.Track
}

type remoteTrackInfo struct {
	trackInfo   TrackInfo
	transceiver *webrtc.RTPTransceiver
	receiver    *webrtc.RTPReceiver
	track       *webrtc.Track
}

func (p *WebRTCTransport) Close() error {
	return p.signaller.Close()
}

func (p *WebRTCTransport) ClientID() string {
	return p.clientID
}

func (p *WebRTCTransport) WriteRTCP(packets []rtcp.Packet) error {
	p.rtcpLog.Printf("[%s] WriteRTCP: %s", p.clientID, packets)
	err := p.peerConnection.WriteRTCP(packets)
	if err == nil {
		prometheusRTCPPacketsSent.Inc()
	}
	return err
}

func (p *WebRTCTransport) CloseChannel() <-chan struct{} {
	return p.signaller.CloseChannel()
}

func (p *WebRTCTransport) WriteRTP(packet *rtp.Packet) (bytes int, err error) {
	p.rtpLog.Printf("[%s] WriteRTP: %s", p.clientID, packet)

	p.mu.Lock()
	defer p.mu.Unlock()

	pta, ok := p.localTracks[packet.SSRC]
	if !ok {
		return 0, fmt.Errorf("Track not found: %d", packet.SSRC)
	}
	if err != nil {
		return 0, err
	}
	err = pta.track.WriteRTP(packet)
	if err == io.ErrClosedPipe {
		// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	prometheusRTPPacketsSent.Inc()
	prometheusRTPPacketsSentBytes.Add(float64(packet.MarshalSize()))
	return packet.MarshalSize(), nil
}

func (p *WebRTCTransport) RemoveTrack(ssrc uint32) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	pta, ok := p.localTracks[ssrc]
	if !ok {
		return fmt.Errorf("Track not found: %d", ssrc)
	}

	err := p.peerConnection.RemoveTrack(pta.sender)
	if err != nil {
		return err
	}

	p.signaller.Negotiate()

	delete(p.localTracks, ssrc)
	return nil
}

func (p *WebRTCTransport) AddTrack(payloadType uint8, ssrc uint32, id string, label string) error {
	track, err := p.peerConnection.NewTrack(payloadType, ssrc, id, label)
	if err != nil {
		return err
	}
	sender, err := p.peerConnection.AddTrack(track)
	if err != nil {
		return err
	}

	if p.signaller.Initiator() {
		p.signaller.Negotiate()
	} else {
		p.signaller.SendTransceiverRequest(track.Kind(), webrtc.RTPTransceiverDirectionRecvonly)
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		for {
			rtcpPackets, err := sender.ReadRTCP()
			if err != nil {
				return
			}
			for _, rtcpPacket := range rtcpPackets {
				p.rtcpLog.Printf("[%s] ReadRTCP: %s", p.clientID, rtcpPacket)
				prometheusRTCPPacketsReceived.Inc()
				p.rtcpCh <- rtcpPacket
			}
		}
	}()

	var transceiver *webrtc.RTPTransceiver
	for _, tr := range p.peerConnection.GetTransceivers() {
		if tr.Sender() == sender {
			transceiver = tr
			break
		}
	}

	trackInfo := TrackInfo{
		SSRC:        track.SSRC(),
		PayloadType: track.PayloadType(),
		ID:          track.ID(),
		Label:       track.Label(),
		Kind:        track.Kind(),
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.localTracks[ssrc] = localTrackInfo{trackInfo, transceiver, sender, track}
	return nil
}

func (p *WebRTCTransport) addRemoteTrack(rti remoteTrackInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.remoteTracks[rti.trackInfo.SSRC] = rti
}

func (p *WebRTCTransport) removeRemoteTrack(ssrc uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.remoteTracks, ssrc)
}

// RemoteTracks returns info about receiving tracks
func (p *WebRTCTransport) RemoteTracks() []TrackInfo {
	p.mu.Lock()
	defer p.mu.Unlock()

	list := make([]TrackInfo, 0, len(p.remoteTracks))
	for _, rti := range p.remoteTracks {
		trackInfo := rti.trackInfo
		trackInfo.Mid = rti.transceiver.Mid()
		list = append(list, trackInfo)
	}
	return list
}

// LocalTracks returns info about sending tracks
func (p *WebRTCTransport) LocalTracks() []TrackInfo {
	p.mu.Lock()
	defer p.mu.Unlock()

	list := make([]TrackInfo, 0, len(p.localTracks))
	for _, lti := range p.localTracks {
		trackInfo := lti.trackInfo
		trackInfo.Mid = lti.transceiver.Mid()
		list = append(list, trackInfo)
	}
	return list
}

func (p *WebRTCTransport) handleTrack(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
	trackInfo := TrackInfo{
		SSRC:        track.SSRC(),
		PayloadType: track.PayloadType(),
		ID:          track.ID(),
		Label:       track.Label(),
		Kind:        track.Kind(),
	}

	p.log.Printf("[%s] Remote track: %d", p.clientID, trackInfo.SSRC)

	start := time.Now()
	prometheusWebRTCTracksTotal.Inc()
	prometheusWebRTCTracksActive.Inc()

	var transceiver *webrtc.RTPTransceiver
	for _, tr := range p.peerConnection.GetTransceivers() {
		if tr.Receiver() == receiver {
			transceiver = tr
			break
		}
	}

	rti := remoteTrackInfo{trackInfo, transceiver, receiver, track}

	p.addRemoteTrack(rti)
	p.trackEventsCh <- TrackEvent{
		TrackInfo: trackInfo,
		Type:      TrackEventTypeAdd,
	}

	p.wg.Add(1)
	go func() {
		defer func() {
			p.removeRemoteTrack(trackInfo.SSRC)
			p.trackEventsCh <- TrackEvent{
				TrackInfo: trackInfo,
				Type:      TrackEventTypeRemove,
			}

			p.wg.Done()

			prometheusWebRTCTracksActive.Dec()
			prometheusWebRTCTracksDuration.Observe(time.Now().Sub(start).Seconds())
		}()

		for {
			pkt, err := track.ReadRTP()
			if err != nil {
				p.log.Printf("[%s] Remote track has ended: %d: %s", p.clientID, trackInfo.SSRC, err)
				return
			}
			prometheusRTPPacketsReceived.Inc()
			prometheusRTPPacketsReceivedBytes.Add(float64(pkt.MarshalSize()))
			p.rtpLog.Printf("[%s] ReadRTP: %s", p.clientID, pkt)
			p.rtpCh <- pkt
		}
	}()
}

func (p *WebRTCTransport) Signal(payload map[string]interface{}) error {
	return p.signaller.Signal(payload)
}

func (p *WebRTCTransport) SignalChannel() <-chan Payload {
	return p.signaller.SignalChannel()
}

func (p *WebRTCTransport) TrackEventsChannel() <-chan TrackEvent {
	return p.trackEventsCh
}

func (p *WebRTCTransport) RTPChannel() <-chan *rtp.Packet {
	return p.rtpCh
}

func (p *WebRTCTransport) RTCPChannel() <-chan rtcp.Packet {
	return p.rtcpCh
}

func (p *WebRTCTransport) MessagesChannel() <-chan webrtc.DataChannelMessage {
	return p.dataTransceiver.MessagesChannel()
}
