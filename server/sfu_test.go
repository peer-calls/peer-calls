package server_test

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/peer-calls/peer-calls/server"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/pionlogger"
	"github.com/peer-calls/peer-calls/server/sfu"
	"github.com/peer-calls/peer-calls/server/test"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"nhooyr.io/websocket"
)

func setupSFUServer(rooms server.RoomManager, jitterBufferEnabled bool) (s *httptest.Server, url string) {
	log := test.NewLogger()

	handler := server.NewSFUHandler(
		log,
		server.NewWSS(log, rooms),
		[]server.ICEServer{},
		server.NetworkConfigSFU{},
		sfu.NewTracksManager(log, jitterBufferEnabled),
	)
	s = httptest.NewServer(handler)
	url = "ws" + strings.TrimPrefix(s.URL, "http") + "/ws/"
	return
}

func TestSFU_ConnectDisconnect(t *testing.T) {
	log := test.NewLogger()

	defer goleak.VerifyNone(t)
	newAdapter := server.NewAdapterFactory(log, server.StoreConfig{})
	defer newAdapter.Close()
	rooms := server.NewAdapterRoomManager(newAdapter.NewAdapter)
	server, url := setupSFUServer(rooms, false)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	wsc := mustDialWS(t, ctx, url)
	err := wsc.Close(websocket.StatusNormalClosure, "")
	require.Nil(t, err, "error closing client socket")
}

func waitForUsersEvent(t *testing.T, ctx context.Context, ch <-chan server.Message) {
	t.Helper()

	for {
		select {
		case msg := <-ch:
			// t.Log("1 msg.Type", msg)
			if msg.Type == server.MessageTypeUsers {
				assert.Equal(t, roomName, msg.Room)
				payload := msg.Payload.(map[string]interface{})
				assert.Equal(t, "__SERVER__", payload["initiator"])
				assert.Equal(t, []interface{}{"__SERVER__"}, payload["peerIds"])
				// assert.Equal(t, map[string]interface{}{clientID: "some-user"}, payload["nicknames"])
				return
			}
		case <-ctx.Done():
			t.Errorf("context timeout: %s", ctx.Err())
			return
		}
	}
}

func startSignalling(
	t *testing.T,
	wsClient *server.Client,
	wsRecvCh <-chan server.Message,
	signaller *server.Signaller,
) {
	t.Helper()
	signalChan := signaller.SignalChannel()

	go func() {
		for msg := range wsRecvCh {
			payload, ok := msg.Payload.(map[string]interface{})

			switch msg.Type {
			case server.MessageTypeSignal:
				require.True(t, ok, "invalid signal msg payload type")
				err := signaller.Signal(payload)
				require.NoError(t, err, "error in receiving signal payload: %w", err)
			case server.MessageTypePubTrack:
				if transport.TrackEventType(payload["type"].(float64)) == transport.TrackEventTypeAdd {
					err := wsClient.Write(server.Message{
						Type: server.MessageTypeSubTrack,
						Payload: map[string]interface{}{
							"type":        transport.TrackEventTypeSub,
							"trackId":     payload["trackId"].(string),
							"pubClientId": payload["pubClientId"].(string),
						},
						Room: msg.Room,
					})
					require.NoError(t, err, "error sending sub_track event to ws: %w", err)
				}

			default:
				// fmt.Println("unhandled ws message", msg.Type, payload)
			}
		}
	}()

	go func() {
		for signal := range signalChan {
			err := wsClient.Write(server.NewMessage("signal", roomName, signal))
			// Sometimes there are late signals e created even after the test has
			// finished successfully, so ignore the errors, but log them.

			if err != nil {
				t.Log(fmt.Errorf("error sending signal to ws: %w", err))
			}
		}
	}()
}

type peerCtx struct {
	pc        *webrtc.PeerConnection
	signaller *server.Signaller
	wsClient  *server.Client
	msg       <-chan server.Message
	close     func() error
}

func createPeerConnection(t *testing.T, ctx context.Context, url string, clientID string) peerCtx {
	t.Helper()

	var peerCtx peerCtx

	var closer test.Closer

	peerCtx.close = closer.Close

	wsc := mustDialWS(t, ctx, url)
	closer.AddFuncErr(func() error {
		return wsc.Close(websocket.StatusNormalClosure, "")
	})

	wsClient := server.NewClientWithID(wsc, clientID)
	msgChan := wsClient.Subscribe(ctx)

	err := wsClient.Write(server.NewMessage("ready", roomName, map[string]interface{}{
		"nickname": "some-user",
	}))
	require.NoError(t, err, "error sending ready message")

	waitForUsersEvent(t, ctx, msgChan)
	require.Nil(t, wsClient.Err())

	var mediaEngine webrtc.MediaEngine
	server.RegisterCodecs(&mediaEngine, false)

	log := test.NewLogger()

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(&mediaEngine),
		webrtc.WithSettingEngine(webrtc.SettingEngine{
			LoggerFactory: pionlogger.NewFactory(log),
		}),
	)

	peerCtx.pc, err = api.NewPeerConnection(webrtc.Configuration{})
	require.Nil(t, err, "error creating peer connection")

	peerCtx.signaller, err = server.NewSignaller(
		log,
		false,
		peerCtx.pc,
		clientID,
		"__SERVER__",
	)
	require.Nil(t, err, "error creating signaller")

	closer.AddFuncErr(peerCtx.signaller.Close) // also closes pc

	startSignalling(t, wsClient, msgChan, peerCtx.signaller)

	return peerCtx
}

func waitPeerConnected(t *testing.T, ctx context.Context, pc *webrtc.PeerConnection) {
	t.Helper()
	waitCh := make(chan struct{})
	// signaller and negotiator do not use this method. if they were, resetting
	// this listener could mess up their functionality. in the future, the whole
	// pc could be wrapped by signaller or some other higher-level struct
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateConnected {
			pc.OnConnectionStateChange(nil)
			close(waitCh)
		}
	})
	wait(t, ctx, waitCh)
}

func sendVideoUntilDone(t *testing.T, done <-chan struct{}, track *webrtc.TrackLocalStaticSample) {
	for {
		select {
		case <-time.After(20 * time.Millisecond):
			assert.NoError(t, track.WriteSample(media.Sample{Data: []byte{0x00}}))
		case <-done:
			return
		}
	}
}

func wait(t *testing.T, ctx context.Context, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
		return
	case <-ctx.Done():
		t.Error("context cancelled:", ctx.Err())
	}
}

func TestSFU_PeerConnection(t *testing.T) {
	log := test.NewLogger()

	defer goleak.VerifyNone(t)
	newAdapter := server.NewAdapterFactory(log, server.StoreConfig{})
	defer newAdapter.Close()
	rooms := server.NewAdapterRoomManager(newAdapter.NewAdapter)
	srv, wsBaseURL := setupSFUServer(rooms, false)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	peerCtx := createPeerConnection(t, ctx, wsBaseURL+roomName+"/"+clientID, clientID)
	defer peerCtx.close()
	waitPeerConnected(t, ctx, peerCtx.pc)
}

func TestSFU_OnTrack(t *testing.T) {
	log := test.NewLogger()

	defer goleak.VerifyNone(t)
	newAdapter := server.NewAdapterFactory(log, server.StoreConfig{})

	log = log.WithNamespaceAppended("test")

	defer newAdapter.Close()
	rooms := server.NewAdapterRoomManager(newAdapter.NewAdapter)
	srv, wsBaseURL := setupSFUServer(rooms, false)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	peerCtx1 := createPeerConnection(t, ctx, wsBaseURL+roomName+"/"+clientID, clientID)
	defer peerCtx1.close()
	waitPeerConnected(t, ctx, peerCtx1.pc)

	fmt.Println("PEER 1 CONNECTED")

	peerCtx2 := createPeerConnection(t, ctx, wsBaseURL+roomName+"/"+clientID2, clientID2)
	defer peerCtx2.close()
	waitPeerConnected(t, ctx, peerCtx2.pc)

	fmt.Println("PEER 2 CONNECTED")

	pc1 := peerCtx1.pc
	pc2 := peerCtx2.pc

	onTrackFired, onTrackFiredDone := context.WithCancel(ctx)
	onTrackEOF, onTrackEOFDone := context.WithCancel(ctx)
	pc2.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log := log.WithCtx(logger.Ctx{"ssrc": remoteTrack.SSRC()})

		log.Info("OnTrack", nil)

		onTrackFiredDone()
		for {
			_, _, err := remoteTrack.ReadRTP()
			fmt.Println("got rtp", err)
			if err != nil {
				log.Info("pc2 remote track ended", nil)
				assert.Equal(t, io.EOF, err, "error reading track")
				onTrackEOFDone()
				return
			}
		}
	})

	localTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: "video/VP8"}, "track-one", "stream-one",
	)
	require.NoError(t, err)

	log.Info("AddTrack start", logger.Ctx{
		"track_id":  localTrack.ID(),
		"stream_id": localTrack.StreamID(),
	})

	sender, err := pc1.AddTrack(localTrack)

	log.Info("AddTrack end", logger.Ctx{
		"track_id":  localTrack.ID(),
		"stream_id": localTrack.StreamID(),
	})

	require.NoError(t, err)
	require.NotNil(t, sender)

	go sendVideoUntilDone(t, onTrackFired.Done(), localTrack)

	log.Info("sending negotiate request (1)", nil)
	wait(t, ctx, peerCtx1.signaller.Negotiate())

	log.Info("waiting for track", nil)

	<-onTrackFired.Done()
	assert.Equal(t, context.Canceled, onTrackFired.Err(), "test timed out")

	log.Info("got track", nil)

	// trigger io.EOF when reading from track2
	assert.NoError(t, pc1.RemoveTrack(sender))

	log.Info("sending negotiate request (2)", nil)
	wait(t, ctx, peerCtx1.signaller.Negotiate())

	<-onTrackEOF.Done()
	assert.Equal(t, context.Canceled, onTrackEOF.Err(), "test timed out")

	log.Info("waiting for peer2 negotiation to be done (3)", nil)
	// server will want to negotiate after track is removed so we wait for negotiation to complete
	wait(t, ctx, peerCtx2.signaller.NegotiationDone())
}
