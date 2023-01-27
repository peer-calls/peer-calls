package cli

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/peer-calls/peer-calls/v4/server/cli/play"
	"github.com/peer-calls/peer-calls/v4/server/codecs"
	"github.com/peer-calls/peer-calls/v4/server/command"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/message"
	"github.com/peer-calls/peer-calls/v4/server/multierr"
	"github.com/peer-calls/peer-calls/v4/server/pionlogger"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/peer-calls/peer-calls/v4/server/uuid"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/spf13/pflag"
	"nhooyr.io/websocket"
)

type playHandler struct {
	args struct {
		config string

		roomURL  string
		nickname string
		insecure bool

		audioMimeType string
		audioFmtp     string
		audioStream   string
		audioSSRC     uint32
		videoMimeType string
		videoFmtp     string
		videoStream   string
		videoSSRC     uint32
	}

	config        server.Config
	codecRegistry *codecs.Registry

	log      logger.Logger
	wg       sync.WaitGroup
	clientID identifiers.ClientID
	roomID   identifiers.RoomID
	wsURL    string

	api         *webrtc.API
	interceptor interceptor.Interceptor
}

// Sample commands. Run ffmpeg in one terminal:
//
//     ffmpeg \
//       -re \
//       -stream_loop -1 \
//       -i "video.mp4" \
//       -an -c:v libvpx -ssrc 1 -payload_type 96 -crf 10 -b:v 1M -cpu-used 5 \
//         -deadline 1 -g 10 -error-resilient 1 -auto-alt-ref 1 -f rtp -max_delay 0 \
//         'rtp://127.0.0.1:50000?localrtcpport=50002&pkt_size=1200' \
//       -vn -c:a libopus -ssrc 2 -b:a 48000 -payload_type 111 -f rtp -max_delay 0 \
//         -application lowdelay 'rtp://127.0.0.1:50004?localrtcpport=50006&pkt_size=1200'
//
// Then, run the play command:
//
//     peer-calls play \
//       --video-stream 'rtp://127.0.0.1:50000?localrtcpport=50002&pkt_size=1200' \
//       --video-ssrc 96 \
//       --video-mime-type video/vp8 \
//       --audio-stream 'rtp://127.0.0.1:50004?localrtcpport=50006&pkt_size=1200' \
//       --audio-ssrc 111 \
//       --audio-mime-type audio/opus \
//       --nickname Player \
//       --room-url http://localhost:3000/call/playroom
//
// And open the browser on the same URL as `--room-url`. You should see the
// video playing after joining.

func (h *playHandler) RegisterFlags(c *command.Command, flags *pflag.FlagSet) {
	flags.StringVarP(&h.args.config, "config", "c", "", "configuration to use")

	flags.StringVarP(&h.args.videoStream, "video-stream", "v", "", "video stream to read RTP from")
	flags.StringVar(&h.args.videoMimeType, "video-mime-type", "video/vp8", "video mime type")
	flags.StringVar(&h.args.videoFmtp, "video-fmtp", "", "video media format parameters")
	flags.Uint32Var(&h.args.videoSSRC, "video-ssrc", 1, "video SSRC")

	flags.StringVarP(&h.args.audioStream, "audio-stream", "a", "", "audio stream to read RTP from")
	flags.StringVar(&h.args.audioMimeType, "audio-mime-type", "audio/opus", "audio mime type")
	flags.StringVar(&h.args.audioFmtp, "audio-fmtp", "", "audio media format parameters")
	flags.Uint32Var(&h.args.audioSSRC, "audio-ssrc", 2, "audio SSRC")

	flags.StringVarP(&h.args.roomURL, "room-url", "r", "http://localhost:3000/call/playroom", "room URL")
	flags.StringVarP(&h.args.nickname, "nickname", "n", "player", "nickname")
	flags.BoolVarP(&h.args.insecure, "insecure", "k", false, "do not validate TLS certificates")
}

func (h *playHandler) configure() (err error) {
	h.codecRegistry = codecs.NewRegistryDefault()

	configFiles := []string{}
	if h.args.config != "" {
		configFiles = append(configFiles, h.args.config)
	}

	h.config, err = server.ReadConfig(configFiles)
	if err != nil {
		return errors.Annotate(err, "read config")
	}

	roomURL, err := url.Parse(h.args.roomURL)
	if err != nil {
		return errors.Trace(err)
	}

	h.clientID = identifiers.ClientID(uuid.New())

	if roomURL.Scheme != "http" && roomURL.Scheme != "https" {
		return errors.Errorf("only http:// or https:// supported, but got: %s", h.args.roomURL)
	}

	roomURL.Scheme = "ws" + strings.TrimPrefix(roomURL.Scheme, "http")

	paths := strings.Split(roomURL.Path, "/")
	h.roomID = identifiers.RoomID(paths[len(paths)-1])

	roomURL.Path = fmt.Sprintf("/ws/%s/%s", h.roomID, h.clientID)

	h.wsURL = roomURL.String()

	mediaEngine1 := server.NewMediaEngine()

	interceptorRegistry1, err := server.NewInterceptorRegistry(mediaEngine1)
	if err != nil {
		h.log.Error("New interceptor registry", errors.Trace(err), nil)
	}

	mediaEngine2 := server.NewMediaEngine()

	interceptorRegistry2, err := server.NewInterceptorRegistry(mediaEngine2)
	if err != nil {
		h.log.Error("New interceptor registry", errors.Trace(err), nil)
	}

	settingEngine := webrtc.SettingEngine{
		LoggerFactory: pionlogger.NewFactory(h.log),
		BufferFactory: nil,
	}

	h.api = webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine1),
		webrtc.WithSettingEngine(settingEngine),
		webrtc.WithInterceptorRegistry(interceptorRegistry1),
	)

	h.interceptor, err = interceptorRegistry2.Build(h.clientID.String())
	if err != nil {
		return errors.Annotate(err, "building interceptors")
	}

	return nil
}

func (h *playHandler) isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func (h *playHandler) startRTPLoop(ctx context.Context, stream *playStream) {
	h.wg.Add(1)

	go func() {
		defer h.wg.Done()

		<-ctx.Done()
		stream.RTPReader.Close()
	}()

	for !h.isDone(ctx) {
		pkt, _, err := stream.RTPReader.ReadRTP()
		if err != nil {
			// h.log.Error("read RTP err", err, nil)

			continue
		}

		if err := stream.Track.WriteRTP(pkt); err != nil {
			h.log.Error("write RTP", err, nil)

			continue
		}
	}
}

func (h *playHandler) startRTCPLoop(ctx context.Context, stream *playStream) {
	h.wg.Add(1)

	go func() {
		defer h.wg.Done()

		<-ctx.Done()
		stream.RTCPReader.Close()
	}()

	for !h.isDone(ctx) {
		// Ensure RTCP interceptor does it work.
		_, _, err := stream.RTCPReader.ReadRTCP()
		if err != nil {
			// h.log.Error("read RTCP err", err, nil)

			continue
		}
	}
}

func (h *playHandler) handleMessages(ctx context.Context, wsClient *server.Client, streams []*playStream) {
	type peerCtx struct {
		pc        *webrtc.PeerConnection
		signaller *server.Signaller
	}

	messagesChan := wsClient.Messages()

	err := wsClient.Write(message.NewReady(h.roomID, message.Ready{
		Nickname: h.args.nickname,
	}))
	if err != nil {
		h.log.Error("unable to send ready message", errors.Trace(err), nil)
	}

	peers := make(map[identifiers.ClientID]*peerCtx)

	defer func() {
		for _, pCtx := range peers {
			pCtx.signaller.Close()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			wsClient.Close(websocket.StatusNormalClosure, "")
			return
		case msg, ok := <-messagesChan:
			if !ok {
				return
			}

			switch msg.Type {
			case message.TypeUsers:
				users := msg.Payload.Users

				initiator := users.Initiator == h.clientID

				pCtxToDelete := make(map[identifiers.ClientID]struct{}, len(users.PeerIDs))

				for _, clientID := range users.PeerIDs {
					pCtxToDelete[clientID] = struct{}{}
				}

				for _, clientID := range users.PeerIDs {
					clientID := clientID
					delete(pCtxToDelete, clientID)

					if _, ok := peers[clientID]; !ok {
						webrtcICEServers := []webrtc.ICEServer{}

						for _, iceServer := range server.GetICEAuthServers(h.config.ICEServers) {
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

						pc, err := h.api.NewPeerConnection(webrtc.Configuration{
							ICEServers: webrtcICEServers,
						})

						if err != nil {
							h.log.Error("Create peer connection", errors.Trace(err), nil)
							continue
						}

						signaller, err := server.NewSignaller(h.log, initiator, pc)
						if err != nil {
							pc.Close()
							h.log.Error("Create signaller connection", errors.Trace(err), nil)
							continue
						}

						peers[clientID] = &peerCtx{
							pc:        pc,
							signaller: signaller,
						}

						h.wg.Add(1)

						signalChan := signaller.SignalChannel()

						go func() {
							defer h.wg.Done()

							for signal := range signalChan {
								userSignal := message.UserSignal{
									PeerID: h.clientID,
									Signal: signal,
								}

								err := wsClient.Write(message.NewSignal(h.roomID, userSignal))
								if err != nil {
									// Sometimes there are late signals created even after the test has
									// finished successfully, so ignore the errors, but log them.
									h.log.Error("send signal to ws", errors.Trace(err), nil)
								}
							}
						}()

						for _, stream := range streams {
							h.log.Info("Add track", logger.Ctx{
								"kind": stream.Track.Kind(),
							})

							stream := stream
							sender, err := pc.AddTrack(stream.Track)
							if err != nil {
								h.log.Error("error adding track", errors.Trace(err), nil)

								continue
							}

							if signaller.Initiator() {
								signaller.Negotiate()
							} else {
								signaller.SendTransceiverRequest(stream.Track.Kind(), webrtc.RTPTransceiverDirectionRecvonly)
							}

							h.wg.Add(1)

							go func() {
								defer h.wg.Done()

								for {
									// ReadRTCP to make interceptors do their jobs.
									packets, _, err := sender.ReadRTCP()
									if err != nil && multierr.Is(err, io.EOF) {
										return
									}

									for _, p := range packets {
										switch p.(type) {
										case *rtcp.PictureLossIndication:
											p = &rtcp.PictureLossIndication{
												SenderSSRC: uint32(stream.OriginalSSRC),
												MediaSSRC:  uint32(stream.OriginalSSRC),
											}

											h.log.Info(fmt.Sprintf("Write RTCP: %s", p), nil)

											// TODO congestion control, debounce.
											err := stream.RTCPWriter.WriteRTCP([]rtcp.Packet{p})
											if err != nil {
												h.log.Error("write RTCP", errors.Trace(err), nil)
											}
										default:
										}
									}
								}
							}()
						}
					}
				}

				for clientID := range pCtxToDelete {
					pCtx, ok := peers[clientID]
					if ok {
						delete(peers, clientID)
						pCtx.signaller.Close()
					}
				}
			case message.TypeSignal:
				peerID := msg.Payload.Signal.PeerID
				pCtx, ok := peers[peerID]
				if !ok {
					h.log.Error("got signal for non-existing peer", nil, logger.Ctx{
						"peer_id": peerID,
					})
				}

				err := pCtx.signaller.Signal(msg.Payload.Signal.Signal)
				if err != nil {
					h.log.Error("signaling error", errors.Trace(err), logger.Ctx{
						"peer_id": peerID,
					})
				}
			}
		}
	}
}

type playStream struct {
	RTCPReader *play.RTCPReader
	RTCPWriter *play.RTCPWriter
	RTPReader  *play.RTPReader
	// Codec      webrtc.RTPCodecCapability
	Track        *webrtc.TrackLocalStaticRTP
	OriginalSSRC webrtc.SSRC
}

func (h *playHandler) newPlayStream(
	stream string,
	codec transport.Codec,
	trackID string,
	streamID string,
	ssrc webrtc.SSRC,
) (*playStream, error) {
	streamURL, err := url.Parse(stream)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if streamURL.Scheme != "rtp" {
		return nil, errors.Errorf("only rtp:// is supported, but got: %s", stream)
	}

	codecParameters, codecMatch := h.codecRegistry.FuzzySearch(codec)
	if codecMatch == codecs.MatchNone {
		return nil, errors.Errorf("codec not found: %s", codec.MimeType)
	}

	track, err := webrtc.NewTrackLocalStaticRTP(codecParameters.RTPCodecCapability, trackID, streamID)
	if err != nil {
		return nil, errors.Annotatef(err, "new track")
	}

	interceptorParams, err := h.codecRegistry.InterceptorParamsForCodec(codec)
	if err != nil {
		return nil, errors.Errorf("codec not found: %s", codec.MimeType)
	}

	q := streamURL.Query()

	pktSize, _ := strconv.Atoi(q.Get("pkt_size"))
	if pktSize == 0 {
		pktSize = 1200
	}

	localRTPPort, err := strconv.Atoi(streamURL.Port())
	if err != nil {
		return nil, errors.Trace(err)
	}

	localRTCPPort, _ := strconv.Atoi(q.Get("rtcpport"))
	if localRTCPPort == 0 {
		localRTCPPort = localRTPPort + 1
	}

	hrkljuz, err := strconv.Atoi(q.Get("localrtcpport"))
	if err != nil {
		return nil, errors.Errorf("no local rtcp port")
	}

	localRTCPAddr := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: localRTCPPort,
		Zone: "",
	}

	remoteRTCPAddr := &net.UDPAddr{
		IP:   net.ParseIP(streamURL.Hostname()),
		Port: hrkljuz,
		Zone: "",
	}

	localRTPAddr := &net.UDPAddr{
		IP:   net.ParseIP(streamURL.Hostname()),
		Port: localRTPPort,
		Zone: "",
	}

	rtcpConn, err := net.DialUDP("udp", localRTCPAddr, remoteRTCPAddr)
	if err != nil {
		return nil, errors.Annotatef(err, "dial RTCP udp")
	}

	rtpConn, err := net.ListenUDP("udp", localRTPAddr)
	if err != nil {
		rtcpConn.Close()

		return nil, errors.Annotatef(err, "dial RTP udp")
	}

	rtcpReader := play.NewRTCPReader(play.RTCPReaderParams{
		Conn:        rtcpConn,
		Interceptor: h.interceptor,
		MTU:         pktSize,
	})

	rtcpWriter := play.NewRTCPWriter(play.RTCPWriterParams{
		Conn:        rtcpConn,
		Interceptor: h.interceptor,
		MTU:         pktSize,
	})

	rtpReader := play.NewRTPReader(play.RTPReaderParams{
		Conn:        rtpConn,
		Interceptor: h.interceptor,
		SSRC:        ssrc,
		Codec: transport.Codec{
			MimeType:    codec.MimeType,
			ClockRate:   codecParameters.ClockRate,
			Channels:    codecParameters.Channels,
			SDPFmtpLine: codecParameters.SDPFmtpLine,
		},
		InterceptorParams: interceptorParams,
		MTU:               pktSize,
	})

	s := playStream{
		RTCPReader:   rtcpReader,
		RTCPWriter:   rtcpWriter,
		RTPReader:    rtpReader,
		Track:        track,
		OriginalSSRC: ssrc,
	}

	return &s, nil
}

func (h *playHandler) Handle(ctx context.Context, args []string) error {
	if err := h.configure(); err != nil {
		return errors.Annotatef(err, "configure")
	}

	var errs multierr.Sync

	streamID := uuid.New()

	streams := make([]*playStream, 0, 2)

	if h.args.audioStream != "" {
		audioCodec := transport.Codec{
			MimeType:    h.args.audioMimeType,
			SDPFmtpLine: h.args.audioFmtp,
		}

		stream, err := h.newPlayStream(
			h.args.audioStream,
			audioCodec,
			"audio",
			streamID,
			webrtc.SSRC(h.args.audioSSRC),
		)
		if err != nil {
			return errors.Annotate(err, "create audio stream")
		}

		streams = append(streams, stream)
	}

	if h.args.videoStream != "" {
		videoCodec := transport.Codec{
			MimeType:    h.args.videoMimeType,
			SDPFmtpLine: h.args.videoFmtp,
		}

		stream, err := h.newPlayStream(
			h.args.videoStream,
			videoCodec,
			"video",
			streamID,
			webrtc.SSRC(h.args.videoSSRC),
		)
		if err != nil {
			return errors.Annotate(err, "create video stream")
		}

		streams = append(streams, stream)
	}

	if len(streams) == 0 {
		return errors.Errorf("no audio or video streams defined")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, stream := range streams {
		stream := stream

		h.wg.Add(1)

		go func() {
			defer h.wg.Done()

			h.startRTPLoop(ctx, stream)
		}()

		h.wg.Add(1)

		go func() {
			defer h.wg.Done()

			h.startRTCPLoop(ctx, stream)
		}()
	}

	ws, _, err := websocket.Dial(ctx, h.wsURL, &websocket.DialOptions{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: h.args.insecure,
				},
			},
		},
	})
	if err != nil {
		return errors.Annotatef(err, "dial WS: %s", h.wsURL)
	}

	wsClient := server.NewClientWithID(ws, h.clientID)

	h.wg.Add(1)

	go func() {
		defer h.wg.Done()
		defer cancel()

		h.handleMessages(ctx, wsClient, streams)
	}()

	h.wg.Wait()

	return errors.Trace(errs.Err())
}

func newPlayCmd(props Props) *command.Command {
	h := &playHandler{
		log: props.Log,
	}

	return command.New(command.Params{
		Name:         "play",
		Desc:         "Play RTP streams",
		FlagRegistry: h,
		Handler:      h,
	})
}
