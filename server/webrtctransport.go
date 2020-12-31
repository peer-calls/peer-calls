package server

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/pionlogger"
	"github.com/peer-calls/peer-calls/server/sfu"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type WebRTCTransportFactory struct {
	log        logger.Logger
	iceServers []ICEServer
	webrtcAPI  *webrtc.API
}

func NewWebRTCTransportFactory(
	log logger.Logger,
	iceServers []ICEServer,
	sfuConfig NetworkConfigSFU,
) *WebRTCTransportFactory {
	allowedInterfaces := map[string]struct{}{}
	for _, iface := range sfuConfig.Interfaces {
		allowedInterfaces[iface] = struct{}{}
	}

	log = log.WithNamespaceAppended("webrtc_transport_factory")

	settingEngine := webrtc.SettingEngine{
		LoggerFactory: pionlogger.NewFactory(log),
	}

	networkTypes := NewNetworkTypes(log, sfuConfig.Protocols)
	settingEngine.SetNetworkTypes(networkTypes)

	if udp := sfuConfig.UDP; udp.PortMin > 0 && udp.PortMax > 0 {
		logCtx := logger.Ctx{
			"port_min": udp.PortMin,
			"port_max": udp.PortMin,
		}

		if err := settingEngine.SetEphemeralUDPPortRange(udp.PortMin, udp.PortMax); err != nil {
			err = errors.Trace(err)
			log.Error("Set epheremal UDP port range", errors.Trace(err), logCtx)
		} else {
			log.Info("Set epheremal UDP port range", logCtx)
		}
	}

	tcpEnabled := false

	for _, networkType := range networkTypes {
		if networkType == webrtc.NetworkTypeTCP4 || networkType == webrtc.NetworkTypeTCP6 {
			tcpEnabled = true

			break
		}
	}

	if tcpEnabled {
		tcpAddr := &net.TCPAddr{
			IP:   net.ParseIP(sfuConfig.TCPBindAddr),
			Port: sfuConfig.TCPListenPort,
			Zone: "",
		}

		logCtx := logger.Ctx{
			"remote_addr": tcpAddr,
		}

		tcpListener, err := net.ListenTCP("tcp", tcpAddr)

		if err != nil {
			log.Error("Start TCP listener", errors.Trace(err), logCtx)
		} else {
			log.Info("Start TCP listener", logCtx)

			logger := settingEngine.LoggerFactory.NewLogger("ice-tcp")
			settingEngine.SetICETCPMux(webrtc.NewICETCPMux(logger, tcpListener, 32))
		}
	}

	if len(allowedInterfaces) > 0 {
		settingEngine.SetInterfaceFilter(func(iface string) bool {
			_, ok := allowedInterfaces[iface]

			return ok
		})
	}

	var mediaEngine webrtc.MediaEngine

	RegisterCodecs(&mediaEngine, sfuConfig.JitterBuffer)

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(settingEngine),
	)

	return &WebRTCTransportFactory{log, iceServers, api}
}

func RegisterCodecs(mediaEngine *webrtc.MediaEngine, jitterBufferEnabled bool) {
	mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	rtcpfb := []webrtc.RTCPFeedback{
		{
			Type: webrtc.TypeRTCPFBGoogREMB,
		},
		// webrtc.RTCPFeedback{
		// 	Type:      webrtc.TypeRTCPFBCCM,
		// 	Parameter: "fir",
		// },

		// https://tools.ietf.org/html/rfc4585#section-4.2
		// "pli" indicates the use of Picture Loss Indication feedback as defined
		// in Section 6.3.1.
		{
			Type:      webrtc.TypeRTCPFBNACK,
			Parameter: "pli",
		},
	}

	if jitterBufferEnabled {
		// The feedback type "nack", without parameters, indicates use of the
		// Generic NACK feedback format as defined in Section 6.2.1.
		rtcpfb = append(rtcpfb, webrtc.RTCPFeedback{
			Type:      webrtc.TypeRTCPFBNACK,
			Parameter: "",
		})
	}

	// s.mediaEngine.RegisterCodec(webrtc.NewRTPH264CodecExt(webrtc.DefaultPayloadTypeH264, 90000, rtcpfb, IOSH264Fmtp))
	// s.mediaEngine.RegisterCodec(webrtc.NewRTPVP9Codec(webrtc.DefaultPayloadTypeVP9, 90000))
	mediaEngine.RegisterCodec(webrtc.NewRTPVP8CodecExt(webrtc.DefaultPayloadTypeVP8, 90000, rtcpfb, ""))
}

type WebRTCTransport struct {
	mu sync.RWMutex
	wg sync.WaitGroup

	log logger.Logger

	clientID string
	roomID   string

	peerConnection  *webrtc.PeerConnection
	signaller       *Signaller
	dataTransceiver *DataTransceiver

	trackEventsCh chan transport.TrackEvent
	rtpCh         chan *rtp.Packet
	rtcpCh        chan rtcp.Packet

	localTracks  map[uint32]localTrack
	remoteTracks map[uint32]remoteTrack
}

var _ transport.Transport = &WebRTCTransport{}

func (f WebRTCTransportFactory) NewWebRTCTransport(roomID, clientID string) (*WebRTCTransport, error) {
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

	// nolint:exhaustivestruct
	webrtcConfig := webrtc.Configuration{
		ICEServers: webrtcICEServers,
	}

	peerConnection, err := f.webrtcAPI.NewPeerConnection(webrtcConfig)
	if err != nil {
		return nil, errors.Annotate(err, "new peer connection")
	}

	return NewWebRTCTransport(f.log, roomID, clientID, true, peerConnection)
}

func NewWebRTCTransport(
	log logger.Logger, roomID, clientID string, initiator bool, peerConnection *webrtc.PeerConnection,
) (*WebRTCTransport, error) {
	log = log.WithNamespaceAppended("webrtc_transport").WithCtx(logger.Ctx{
		"client_id": clientID,
	})

	closePeer := func(reason error) error {
		var errs MultiErrorHandler

		errs.Add(reason)

		err := peerConnection.Close()
		if err != nil {
			errs.Add(errors.Annotatef(err, "close peer connection"))
		}

		return errors.Trace(errs.Err())
	}

	var (
		dataChannel *webrtc.DataChannel
		err         error
	)

	if initiator {
		// need to do this to connect with simple peer
		// only when we are the initiator
		dataChannel, err = peerConnection.CreateDataChannel("data", nil)
		if err != nil {
			return nil, closePeer(errors.Annotate(err, "create data channel"))
		}
	}

	dataTransceiver := NewDataTransceiver(log, clientID, dataChannel, peerConnection)

	signaller, err := NewSignaller(
		log,
		initiator,
		peerConnection,
		localPeerID,
		clientID,
	)

	peerConnection.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		log.Info("ICE gathering state changed", logger.Ctx{
			"state": state,
		})
	})

	if err != nil {
		return nil, closePeer(errors.Annotate(err, "initialize signaller"))
	}

	transport := &WebRTCTransport{
		log: log,

		clientID:        clientID,
		signaller:       signaller,
		peerConnection:  peerConnection,
		dataTransceiver: dataTransceiver,

		trackEventsCh: make(chan transport.TrackEvent),
		rtpCh:         make(chan *rtp.Packet),
		rtcpCh:        make(chan rtcp.Packet),

		localTracks:  map[uint32]localTrack{},
		remoteTracks: map[uint32]remoteTrack{},
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

type localTrack struct {
	trackInfo   transport.TrackInfo
	transceiver *webrtc.RTPTransceiver
	sender      *webrtc.RTPSender
	track       *webrtc.Track
}

type remoteTrack struct {
	trackInfo   transport.TrackInfo
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

func (p *WebRTCTransport) Type() transport.Type {
	return transport.TypeWebRTC
}

func (p *WebRTCTransport) WriteRTCP(packets []rtcp.Packet) error {
	p.log.Trace("WriteRTCP", logger.Ctx{
		"packets": packets,
	})

	err := p.peerConnection.WriteRTCP(packets)
	if err == nil {
		prometheusRTCPPacketsSent.Inc()
	}

	return errors.Annotate(err, "write rtcp")
}

func (p *WebRTCTransport) CloseChannel() <-chan struct{} {
	return p.signaller.CloseChannel()
}

func (p *WebRTCTransport) WriteRTP(packet *rtp.Packet) (bytes int, err error) {
	p.log.Trace("WriteRTP", logger.Ctx{
		"packet": packet,
	})

	p.mu.RLock()
	pta, ok := p.localTracks[packet.SSRC]
	p.mu.RUnlock()

	if !ok {
		return 0, errors.Errorf("track %d not found", packet.SSRC)
	}

	err = pta.track.WriteRTP(packet)
	if errIs(err, io.ErrClosedPipe) {
		// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
		return 0, nil
	}

	if err != nil {
		return 0, errors.Annotate(err, "write rtp")
	}

	prometheusRTPPacketsSent.Inc()
	prometheusRTPPacketsSentBytes.Add(float64(packet.MarshalSize()))

	return packet.MarshalSize(), nil
}

func (p *WebRTCTransport) RemoveTrack(ssrc uint32) error {
	p.mu.Lock()

	pta, ok := p.localTracks[ssrc]
	if ok {
		delete(p.localTracks, ssrc)
	}

	p.mu.Unlock()

	if !ok {
		return errors.Errorf("track %d not found", ssrc)
	}

	err := p.peerConnection.RemoveTrack(pta.sender)
	if err != nil {
		return errors.Annotate(err, "remove track")
	}

	p.signaller.Negotiate()

	return nil
}

func (p *WebRTCTransport) AddTrack(t transport.Track) error {
	track, err := p.peerConnection.NewTrack(t.PayloadType(), t.SSRC(), t.ID(), t.Label())
	if err != nil {
		return errors.Annotate(err, "new track")
	}

	sender, err := p.peerConnection.AddTrack(track)
	if err != nil {
		return errors.Annotate(err, "add track")
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
				p.log.Trace("ReadRTCP", logger.Ctx{
					"packet": rtcpPacket,
				})
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

	trackInfo := transport.TrackInfo{
		Track: t,
		Kind:  track.Kind(),
		Mid:   "",
	}

	p.mu.Lock()
	p.localTracks[t.SSRC()] = localTrack{trackInfo, transceiver, sender, track}
	p.mu.Unlock()

	return nil
}

func (p *WebRTCTransport) addRemoteTrack(rti remoteTrack) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.remoteTracks[rti.trackInfo.Track.SSRC()] = rti
}

func (p *WebRTCTransport) removeRemoteTrack(ssrc uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.remoteTracks, ssrc)
}

// RemoteTracks returns info about receiving tracks
func (p *WebRTCTransport) RemoteTracks() []transport.TrackInfo {
	p.mu.Lock()
	defer p.mu.Unlock()

	list := make([]transport.TrackInfo, 0, len(p.remoteTracks))

	for _, rti := range p.remoteTracks {
		trackInfo := rti.trackInfo
		trackInfo.Mid = rti.transceiver.Mid()
		list = append(list, trackInfo)
	}

	return list
}

// LocalTracks returns info about sending tracks
func (p *WebRTCTransport) LocalTracks() []transport.TrackInfo {
	p.mu.Lock()
	defer p.mu.Unlock()

	list := make([]transport.TrackInfo, 0, len(p.localTracks))

	for _, lti := range p.localTracks {
		trackInfo := lti.trackInfo
		trackInfo.Mid = lti.transceiver.Mid()
		list = append(list, trackInfo)
	}

	return list
}

func (p *WebRTCTransport) handleTrack(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
	trackInfo := transport.TrackInfo{
		Track: sfu.NewUserTrack(
			transport.NewSimpleTrack(track.PayloadType(), track.SSRC(), track.ID(), track.Label()),
			p.clientID,
			p.roomID,
		),
		Kind: track.Kind(),
		Mid:  "",
	}

	log := p.log.WithCtx(logger.Ctx{
		"ssrc": trackInfo.Track.SSRC(),
	})

	log.Info("Remote track", nil)

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

	rti := remoteTrack{trackInfo, transceiver, receiver, track}

	p.addRemoteTrack(rti)
	p.trackEventsCh <- transport.TrackEvent{
		TrackInfo: trackInfo,
		Type:      transport.TrackEventTypeAdd,
		ClientID:  p.clientID,
	}

	p.wg.Add(1)

	go func() {
		defer func() {
			p.removeRemoteTrack(trackInfo.Track.SSRC())
			p.trackEventsCh <- transport.TrackEvent{
				TrackInfo: trackInfo,
				Type:      transport.TrackEventTypeRemove,
				ClientID:  p.clientID,
			}

			p.wg.Done()

			prometheusWebRTCTracksActive.Dec()
			prometheusWebRTCTracksDuration.Observe(time.Since(start).Seconds())
		}()

		for {
			pkt, err := track.ReadRTP()
			if err != nil {
				log.Error("Read RTP", errors.Trace(err), nil)

				return
			}

			prometheusRTPPacketsReceived.Inc()
			prometheusRTPPacketsReceivedBytes.Add(float64(pkt.MarshalSize()))

			log.Trace("ReadRTP", logger.Ctx{
				"packet": pkt,
			})
			p.rtpCh <- pkt
		}
	}()
}

func (p *WebRTCTransport) Signal(payload map[string]interface{}) error {
	err := p.signaller.Signal(payload)

	return errors.Annotate(err, "signal")
}

func (p *WebRTCTransport) SignalChannel() <-chan Payload {
	return p.signaller.SignalChannel()
}

func (p *WebRTCTransport) TrackEventsChannel() <-chan transport.TrackEvent {
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

func (p *WebRTCTransport) Send(message webrtc.DataChannelMessage) <-chan error {
	return p.dataTransceiver.Send(message)
}
