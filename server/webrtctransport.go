package server

import (
	"net"
	"strings"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/codecs"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/message"
	"github.com/peer-calls/peer-calls/v4/server/pionlogger"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

type WebRTCTransportFactory struct {
	log           logger.Logger
	iceServers    []ICEServer
	codecRegistry *codecs.Registry
	settingEngine webrtc.SettingEngine
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

	return &WebRTCTransportFactory{log, iceServers, registry, settingEngine}
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
}

type WebRTCTransport struct {
	mu sync.RWMutex

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
	// Something odd is happneing in pion/webrtc.  So to keep this clean, we
	// create a new webrtc.MediaEngine, interceptor.Registry and webrtc.API every
	// time.
	mediaEngine := NewMediaEngine()

	interceptorRegistry, err := NewInterceptorRegistry(mediaEngine)
	if err != nil {
		f.log.Error("New interceptor registry", errors.Trace(err), nil)
	}

	api := webrtc.NewAPI(
		// TODO the documenet for this method says that mediaEngine can be changed
		// after the engine is passed to the API. Perhaps we should keep a separate
		// mediaEngine for each peer connection?
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(f.settingEngine),
		webrtc.WithInterceptorRegistry(interceptorRegistry),
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

		localTracks: map[identifiers.TrackID]localTrack{},

		remoteTracksChannel: make(chan transport.TrackRemoteWithRTCPReader),
	}
	peerConnection.OnTrack(transport.handleTrack)

	go func() {
		// wait for peer connection to be closed
		<-signaller.Done()
		peerConnection.OnTrack(nil)
		transport.dataTransceiver.Close()
	}()
	return transport, nil
}

type localTrack struct {
	trackInfo   transport.TrackWithMID
	transceiver *webrtc.RTPTransceiver
	sender      *webrtc.RTPSender
	track       *webrtc.TrackLocalStaticRTP
}

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

	return errors.Annotate(err, "write rtcp")
}

func (p *WebRTCTransport) Done() <-chan struct{} {
	return p.signaller.Done()
}

func (p *WebRTCTransport) RemoteTracksChannel() <-chan transport.TrackRemoteWithRTCPReader {
	return p.remoteTracksChannel
}

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

	codecParameters, _ := p.codecRegistry.FuzzySearch(codec)

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

	// TODO prometheus, move this to pubsub.

	// prometheusWebRTCTracksTotal.Inc()
	// prometheusWebRTCTracksActive.Inc()

	// 		prometheusWebRTCTracksActive.Dec()
	// 		prometheusWebRTCTracksDuration.Observe(time.Since(start).Seconds())
}

func (p *WebRTCTransport) Signal(signal message.Signal) error {
	err := p.signaller.Signal(signal)

	return errors.Annotate(err, "signal")
}

func (p *WebRTCTransport) SignalChannel() <-chan message.Signal {
	return p.signaller.SignalChannel()
}

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
