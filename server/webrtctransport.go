package server

import (
	"net"
	"strings"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/codecs"
	"github.com/peer-calls/peer-calls/server/identifiers"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/message"
	"github.com/peer-calls/peer-calls/server/pionlogger"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

type WebRTCTransportFactory struct {
	log                 logger.Logger
	iceServers          []ICEServer
	codecRegistry       *codecs.Registry
	interceptorRegistry *interceptor.Registry
	mediaEngine         *webrtc.MediaEngine
	settingEngine       webrtc.SettingEngine
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
		BufferFactory: nil,
	}

	networkTypes := NewNetworkTypes(log, sfuConfig.Protocols)
	settingEngine.SetNetworkTypes(networkTypes)

	if udp := sfuConfig.UDP; udp.PortMin > 0 && udp.PortMax > 0 {
		logCtx := logger.Ctx{
			"port_min": udp.PortMin,
			"port_max": udp.PortMax,
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

	registry := codecs.NewRegistryDefault()

	if len(allowedInterfaces) > 0 {
		settingEngine.SetInterfaceFilter(func(iface string) bool {
			_, ok := allowedInterfaces[iface]

			return ok
		})
	}

	mediaEngine := NewMediaEngine()

	interceptorRegistry, err := NewInterceptorRegistry(mediaEngine)
	if err != nil {
		log.Error("New interceptor registry", errors.Trace(err), nil)
	}

	return &WebRTCTransportFactory{log, iceServers, registry, interceptorRegistry, mediaEngine, settingEngine}
}

func NewMediaEngine() *webrtc.MediaEngine {
	var mediaEngine webrtc.MediaEngine

	registry := codecs.NewRegistryDefault()

	RegisterCodecs(&mediaEngine, registry)

	return &mediaEngine
}

func NewInterceptorRegistry(mediaEngine *webrtc.MediaEngine) (*interceptor.Registry, error) {
	interceptorRegistry := &interceptor.Registry{}

	if err := webrtc.RegisterDefaultInterceptors(mediaEngine, interceptorRegistry); err != nil {
		return nil, errors.Annotatef(err, "registering default interceptors")
	}

	return interceptorRegistry, nil
}

func RegisterCodecs(mediaEngine *webrtc.MediaEngine, registry *codecs.Registry) {
	// TODO handle errors gracefully.

	for _, codec := range registry.Audio.CodecParameters {
		err := mediaEngine.RegisterCodec(codec, webrtc.RTPCodecTypeAudio)
		if err != nil {
			panic(err)
		}
	}

	for _, codec := range registry.Video.CodecParameters {
		err := mediaEngine.RegisterCodec(codec, webrtc.RTPCodecTypeVideo)
		if err != nil {
			panic(err)
		}
	}

	for _, ext := range registry.Audio.HeaderExtensions {
		if err := mediaEngine.RegisterHeaderExtension(
			webrtc.RTPHeaderExtensionCapability{
				URI: ext.Parameter.URI,
			},
			webrtc.RTPCodecTypeAudio,
			ext.AllowedDirections...,
		); err != nil {
			panic(err)
		}
	}

	for _, ext := range registry.Video.HeaderExtensions {
		if err := mediaEngine.RegisterHeaderExtension(
			webrtc.RTPHeaderExtensionCapability{
				URI: ext.Parameter.URI,
			},
			webrtc.RTPCodecTypeAudio,
			ext.AllowedDirections...,
		); err != nil {
			panic(err)
		}
	}

	// rtcpfb := []webrtc.RTCPFeedback{
	// 	{
	// 		Type: webrtc.TypeRTCPFBGoogREMB,
	// 	},
	// 	// webrtc.RTCPFeedback{
	// 	// 	Type:      webrtc.TypeRTCPFBCCM,
	// 	// 	Parameter: "fir",
	// 	// },

	// 	// https://tools.ietf.org/html/rfc4585#section-4.2
	// 	// "pli" indicates the use of Picture Loss Indication feedback as defined
	// 	// in Section 6.3.1.
	// 	{
	// 		Type:      webrtc.TypeRTCPFBNACK,
	// 		Parameter: "pli",
	// 	},
	// }

	// if jitterBufferEnabled {
	// 	// The feedback type "nack", without parameters, indicates use of the
	// 	// Generic NACK feedback format as defined in Section 6.2.1.
	// 	rtcpfb = append(rtcpfb, webrtc.RTCPFeedback{
	// 		Type:      webrtc.TypeRTCPFBNACK,
	// 		Parameter: "",
	// 	})
	// }
}

type WebRTCTransport struct {
	mu sync.RWMutex
	wg sync.WaitGroup

	log logger.Logger

	clientID identifiers.ClientID
	peerID   identifiers.PeerID

	peerConnection  *webrtc.PeerConnection
	signaller       *Signaller
	dataTransceiver *DataTransceiver

	codecRegistry *codecs.Registry

	remoteTracksChannel chan transport.TrackRemoteWithRTCPReader

	localTracks map[identifiers.TrackID]localTrack
}

func (f WebRTCTransportFactory) NewWebRTCTransport(
	roomID identifiers.RoomID,
	clientID identifiers.ClientID,
	peerID identifiers.PeerID,
) (*WebRTCTransport, error) {
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

	// webrtc.PeerConnection.Close will close the intercetpor of the whole API.
	// So to keep this clean, we create a new api every time.
	api := webrtc.NewAPI(
		// TODO the documenet for this method says that mediaEngine can be changed
		// after the engine is passed to the API. Perhaps we should keep a separate
		// mediaEngine for each peer connection?
		webrtc.WithMediaEngine(f.mediaEngine),
		webrtc.WithSettingEngine(f.settingEngine),
		webrtc.WithInterceptorRegistry(f.interceptorRegistry),
	)

	peerConnection, err := api.NewPeerConnection(webrtcConfig)
	if err != nil {
		return nil, errors.Annotate(err, "new peer connection")
	}

	return NewWebRTCTransport(f.log, roomID, clientID, peerID, true, peerConnection, f.codecRegistry)
}

func NewWebRTCTransport(
	log logger.Logger,
	roomID identifiers.RoomID,
	clientID identifiers.ClientID,
	peerID identifiers.PeerID,
	initiator bool,
	peerConnection *webrtc.PeerConnection,
	codecRegistry *codecs.Registry,
) (*WebRTCTransport, error) {
	log = log.WithNamespaceAppended("webrtc_transport").WithCtx(logger.Ctx{
		"client_id": clientID,
		"room_id":   roomID,
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
		peerID:          peerID,
		signaller:       signaller,
		peerConnection:  peerConnection,
		dataTransceiver: dataTransceiver,

		codecRegistry: codecRegistry,

		// trackEventsCh: make(chan transport.TrackEvent),
		// rtpCh:         make(chan *rtp.Packet),
		// rtcpCh:        make(chan []rtcp.Packet),

		localTracks: map[identifiers.TrackID]localTrack{},
		// remoteTracks: map[identifiers.TrackID]remoteTrack{},

		remoteTracksChannel: make(chan transport.TrackRemoteWithRTCPReader),
	}
	peerConnection.OnTrack(transport.handleTrack)

	go func() {
		// wait for peer connection to be closed
		<-signaller.Done()
		// do not close channels before all writing goroutines exit
		transport.wg.Wait()
		transport.dataTransceiver.Close()
		close(transport.remoteTracksChannel)
		// close(transport.rtpCh)
		// close(transport.rtcpCh)
		// close(transport.trackEventsCh)
	}()
	return transport, nil
}

type localTrack struct {
	trackInfo   transport.TrackWithMID
	transceiver *webrtc.RTPTransceiver
	sender      *webrtc.RTPSender
	track       *webrtc.TrackLocalStaticRTP
}

// type remoteTrack struct {
// 	trackInfo   transport.TrackWithMID
// 	transceiver *webrtc.RTPTransceiver
// 	receiver    *webrtc.RTPReceiver
// 	track       *webrtc.TrackRemote
// }

func (p *WebRTCTransport) Close() error {
	return p.signaller.Close()
}

func (p *WebRTCTransport) ClientID() identifiers.ClientID {
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

func (p *WebRTCTransport) Done() <-chan struct{} {
	return p.signaller.Done()
}

func (p *WebRTCTransport) RemoteTracksChannel() <-chan transport.TrackRemoteWithRTCPReader {
	return p.remoteTracksChannel
}

// func (p *WebRTCTransport) WriteRTP(packet *rtp.Packet) (bytes int, err error) {
// 	p.log.Trace("WriteRTP", logger.Ctx{
// 		"packet": packet,
// 	})

// 	p.mu.RLock()
// 	pta, ok := p.localTracks[packet.TrackID]
// 	p.mu.RUnlock()

// 	if !ok {
// 		return 0, errors.Errorf("track %s not found", packet.TrackID)
// 	}

// 	err = pta.track.WriteRTP(packet)
// 	if errIs(err, io.ErrClosedPipe) {
// 		// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
// 		return 0, nil
// 	}

// 	if err != nil {
// 		return 0, errors.Annotate(err, "write rtp")
// 	}

// 	prometheusRTPPacketsSent.Inc()
// 	prometheusRTPPacketsSentBytes.Add(float64(packet.MarshalSize()))

// 	return packet.MarshalSize(), nil
// }

func (p *WebRTCTransport) RemoveTrack(trackID identifiers.TrackID) error {
	p.mu.Lock()

	pta, ok := p.localTracks[trackID]
	if ok {
		delete(p.localTracks, trackID)
	}

	p.mu.Unlock()

	if !ok {
		return errors.Errorf("track %s not found", trackID)
	}

	err := p.peerConnection.RemoveTrack(pta.sender)
	if err != nil {
		return errors.Annotate(err, "remove track")
	}

	// TODO I don't think this would be necessary if we used
	// on negotiation needed callback.
	p.signaller.Negotiate()

	return nil
}

var _ transport.Transport = &WebRTCTransport{}

func (p *WebRTCTransport) AddTrack(t transport.Track) (transport.TrackLocal, transport.RTCPReader, error) {
	codec := t.Codec()

	var rtcpFeedback []webrtc.RTCPFeedback

	codecParameters, _ := p.codecRegistry.FindByMimeType(codec.MimeType)

	if strings.HasPrefix(codec.MimeType, "video/") {
		rtcpFeedback = codecParameters.RTCPFeedback
	}

	capability := webrtc.RTPCodecCapability{
		MimeType:     codec.MimeType,
		ClockRate:    codec.ClockRate,
		Channels:     codec.Channels,
		SDPFmtpLine:  codec.SDPFmtpLine,
		RTCPFeedback: rtcpFeedback,
	}

	trackID := t.TrackID()

	track, err := webrtc.NewTrackLocalStaticRTP(capability, trackID.ID, trackID.StreamID)
	if err != nil {
		return nil, nil, errors.Annotate(err, "new track")
	}

	sender, err := p.peerConnection.AddTrack(track)
	if err != nil {
		return nil, nil, errors.Annotate(err, "add track")
	}

	if p.signaller.Initiator() {
		p.signaller.Negotiate()
	} else {
		p.signaller.SendTransceiverRequest(track.Kind(), webrtc.RTPTransceiverDirectionRecvonly)
	}

	// p.wg.Add(1)

	// go func() {
	// 	defer p.wg.Done()

	// 	for {
	// 		rtcpPackets, _, err := sender.ReadRTCP()
	// 		if err != nil {
	// 			return
	// 		}

	// 		if p.log.IsLevelEnabled(logger.LevelTrace) {
	// 			for _, rtcpPacket := range rtcpPackets {
	// 				p.log.Trace("ReadRTCP", logger.Ctx{
	// 					"packet": rtcpPacket,
	// 				})
	// 			}
	// 		}

	// 		prometheusRTCPPacketsReceived.Add(float64(len(rtcpPackets)))

	// 		p.rtcpCh <- rtcpPackets
	// 	}
	// }()

	var transceiver *webrtc.RTPTransceiver

	for _, tr := range p.peerConnection.GetTransceivers() {
		if tr.Sender() == sender {
			transceiver = tr

			break
		}
	}

	mid := transceiver.Mid()
	_ = mid

	trackInfo := transport.NewTrackWithMID(t, mid)

	p.mu.Lock()
	p.localTracks[t.TrackID()] = localTrack{trackInfo, transceiver, sender, track}
	p.mu.Unlock()

	tt := LocalTrack{
		TrackLocalStaticRTP: track,
		track:               t,
	}

	return tt, sender, nil
}

// func (p *WebRTCTransport) addRemoteTrack(rti remoteTrack) {
// 	p.mu.Lock()
// 	defer p.mu.Unlock()

// 	p.remoteTracks[rti.trackInfo.Track.TrackID()] = rti
// }

// func (p *WebRTCTransport) removeRemoteTrack(trackID identifiers.TrackID) {
// 	p.mu.Lock()
// 	defer p.mu.Unlock()

// 	delete(p.remoteTracks, trackID)
// }

// // RemoteTracks returns info about receiving tracks
// func (p *WebRTCTransport) RemoteTracks() []transport.TrackInfo {
// 	p.mu.Lock()
// 	defer p.mu.Unlock()

// 	list := make([]transport.TrackInfo, 0, len(p.remoteTracks))

// 	for _, rti := range p.remoteTracks {
// 		trackInfo := rti.trackInfo
// 		trackInfo.Mid = rti.transceiver.Mid()
// 		list = append(list, trackInfo)
// 	}

// 	return list
// }

// LocalTracks returns info about sending tracks
func (p *WebRTCTransport) LocalTracks() []transport.TrackWithMID {
	p.mu.Lock()
	defer p.mu.Unlock()

	list := make([]transport.TrackWithMID, 0, len(p.localTracks))

	for _, lti := range p.localTracks {
		// It is important to reread the Mid in case transceiver got reassigned.
		list = append(list, transport.NewTrackWithMID(lti.trackInfo, lti.transceiver.Mid()))
	}

	return list
}

func (p *WebRTCTransport) handleTrack(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
	rtpCodecParameters := track.Codec()

	codec := transport.Codec{
		MimeType:    rtpCodecParameters.MimeType,
		ClockRate:   rtpCodecParameters.ClockRate,
		Channels:    rtpCodecParameters.Channels,
		SDPFmtpLine: rtpCodecParameters.SDPFmtpLine,
	}

	t := RemoteTrack{
		TrackRemote: track,
		track:       transport.NewSimpleTrack(track.ID(), track.StreamID(), codec, p.peerID),
	}

	trwr := transport.TrackRemoteWithRTCPReader{
		TrackRemote: t,
		RTCPReader:  receiver,
	}

	select {
	case p.remoteTracksChannel <- trwr:
	case <-p.signaller.Done():
	}

	// mimeType := track.Codec().MimeType
	// _ = mimeType // FIXME

	// trackInfo := transport.TrackInfo{
	// 	Track: transport.NewSimpleTrack(
	// 		p.clientID, mimeType, track.ID(), track.StreamID(),
	// 	),
	// 	Mid: "",
	// }

	// log := p.log.WithCtx(logger.Ctx{
	// 	"track_id":  trackInfo.Track.ID(),
	// 	"stream_id": trackInfo.Track.StreamID(),
	// })

	// log.Info("Remote track", nil)

	// start := time.Now()

	// prometheusWebRTCTracksTotal.Inc()
	// prometheusWebRTCTracksActive.Inc()

	// var transceiver *webrtc.RTPTransceiver

	// for _, tr := range p.peerConnection.GetTransceivers() {
	// 	if tr.Receiver() == receiver {
	// 		transceiver = tr

	// 		break
	// 	}
	// }

	// rti := remoteTrack{trackInfo, transceiver, receiver, track}

	// p.addRemoteTrack(rti)
	// p.trackEventsCh <- transport.TrackEvent{
	// 	TrackInfo: trackInfo,
	// 	Type:      transport.TrackEventTypeAdd,
	// 	ClientID:  p.clientID,
	// }

	// p.wg.Add(1)

	// go func() {
	// 	defer func() {
	// 		p.removeRemoteTrack(trackInfo.Track.TrackID())
	// 		p.trackEventsCh <- transport.TrackEvent{
	// 			TrackInfo: trackInfo,
	// 			Type:      transport.TrackEventTypeRemove,
	// 			ClientID:  p.clientID,
	// 		}

	// 		p.wg.Done()

	// 		prometheusWebRTCTracksActive.Dec()
	// 		prometheusWebRTCTracksDuration.Observe(time.Since(start).Seconds())
	// 	}()

	// 	for {
	// 		pkt, _, err := track.ReadRTP()
	// 		if err != nil {
	// 			log.Error("Read RTP", errors.Trace(err), nil)

	// 			return
	// 		}

	// 		prometheusRTPPacketsReceived.Inc()
	// 		prometheusRTPPacketsReceivedBytes.Add(float64(pkt.MarshalSize()))

	// 		log.Trace("ReadRTP", logger.Ctx{
	// 			"packet": pkt,
	// 		})
	// 		p.rtpCh <- pkt
	// 	}
	// }()
}

func (p *WebRTCTransport) Signal(signal message.Signal) error {
	err := p.signaller.Signal(signal)

	return errors.Annotate(err, "signal")
}

func (p *WebRTCTransport) SignalChannel() <-chan message.Signal {
	return p.signaller.SignalChannel()
}

// func (p *WebRTCTransport) TrackEventsChannel() <-chan transport.TrackEvent {
// 	return p.trackEventsCh
// }

// func (p *WebRTCTransport) RTPChannel() <-chan *rtp.Packet {
// 	return p.rtpCh
// }

// func (p *WebRTCTransport) RTCPChannel() <-chan []rtcp.Packet {
// 	return p.rtcpCh
// }

func (p *WebRTCTransport) MessagesChannel() <-chan webrtc.DataChannelMessage {
	return p.dataTransceiver.MessagesChannel()
}

func (p *WebRTCTransport) Send(message webrtc.DataChannelMessage) <-chan error {
	return p.dataTransceiver.Send(message)
}

type LocalTrack struct {
	*webrtc.TrackLocalStaticRTP
	track transport.Track
}

func (t LocalTrack) Track() transport.Track {
	return t.track
}

type RemoteTrack struct {
	*webrtc.TrackRemote
	track transport.Track
}

func (t RemoteTrack) Track() transport.Track {
	return t.track
}
