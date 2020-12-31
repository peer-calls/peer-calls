package server_test

import (
	"context"
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
	t *testing.T, wsClient *server.Client, wsRecvCh <-chan server.Message, signaller *server.Signaller,
) {
	t.Helper()
	signalChan := signaller.SignalChannel()

	go func() {
		for msg := range wsRecvCh {
			if msg.Type == "signal" {
				payload, ok := msg.Payload.(map[string]interface{})
				require.True(t, ok, "invalid signal msg payload type")
				err := signaller.Signal(payload)
				require.NoError(t, err, "error in receiving signal payload: %w", err)
			}
		}
	}()

	go func() {
		for signal := range signalChan {
			err := wsClient.Write(server.NewMessage("signal", roomName, signal))
			require.NoError(t, err, "error sending signal to ws: %w", err)
		}
	}()
}

func createPeerConnection(t *testing.T, ctx context.Context, url string, clientID string) (pc *webrtc.PeerConnection, signaller *server.Signaller, cleanup func() error) {
	t.Helper()
	var closer test.Closer
	cleanup = closer.Close

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
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(webrtc.SettingEngine{
			LoggerFactory: pionlogger.NewFactory(log),
		}),
	)

	pc, err = api.NewPeerConnection(webrtc.Configuration{})
	require.Nil(t, err, "error creating peer connection")

	signaller, err = server.NewSignaller(
		log,
		false,
		pc,
		clientID,
		"__SERVER__",
	)
	require.Nil(t, err, "error creating signaller")

	closer.AddFuncErr(signaller.Close) // also closes pc

	startSignalling(t, wsClient, msgChan, signaller)

	return pc, signaller, cleanup
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

func sendVideoUntilDone(t *testing.T, done <-chan struct{}, track *webrtc.Track) {
	for {
		select {
		case <-time.After(20 * time.Millisecond):
			assert.NoError(t, track.WriteSample(media.Sample{Data: []byte{0x00}, Samples: 1}))
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
	pc, _, cleanup := createPeerConnection(t, ctx, wsBaseURL+roomName+"/"+clientID, clientID)
	defer cleanup()
	waitPeerConnected(t, ctx, pc)
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

	pc1, signaller1, cleanup := createPeerConnection(t, ctx, wsBaseURL+roomName+"/"+clientID, clientID)
	defer cleanup()
	waitPeerConnected(t, ctx, pc1)

	pc2, signaller2, cleanup := createPeerConnection(t, ctx, wsBaseURL+roomName+"/"+clientID2, clientID2)
	defer cleanup()
	waitPeerConnected(t, ctx, pc2)

	onTrackFired, onTrackFiredDone := context.WithCancel(ctx)
	onTrackEOF, onTrackEOFDone := context.WithCancel(ctx)
	pc2.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		log := log.WithCtx(logger.Ctx{"ssrc": remoteTrack.SSRC()})

		log.Info("OnTrack", nil)

		onTrackFiredDone()
		for {
			_, err := remoteTrack.ReadRTP()
			if remoteTrack != nil {
				log.Info("pc2 remote track ended", nil)
				assert.Equal(t, io.EOF, err, "error reading track")
				onTrackEOFDone()
				return
			}
		}
	})

	localTrack, err := pc1.NewTrack(webrtc.DefaultPayloadTypeVP8, 12345, "track-one", "stream-one")
	require.NoError(t, err)

	log.Info("AddTrack start", logger.Ctx{
		"ssrc": localTrack.SSRC(),
	})

	sender, err := pc1.AddTrack(localTrack)

	log.Info("AddTrack end", logger.Ctx{
		"ssrc": localTrack.SSRC(),
	})

	require.NoError(t, err)
	require.NotNil(t, sender)

	go sendVideoUntilDone(t, onTrackFired.Done(), localTrack)

	log.Info("sending negotiate request (1)", nil)
	wait(t, ctx, signaller1.Negotiate())

	<-onTrackFired.Done()
	assert.Equal(t, context.Canceled, onTrackFired.Err(), "test timed out")

	// trigger io.EOF when reading from track2
	assert.NoError(t, pc1.RemoveTrack(sender))

	log.Info("sending negotiate request (2)", nil)
	wait(t, ctx, signaller1.Negotiate())

	<-onTrackEOF.Done()
	assert.Equal(t, context.Canceled, onTrackEOF.Err(), "test timed out")

	log.Info("waiting for peer2 negotiation to be done (3)", nil)
	// server will want to negotiate after track is removed so we wait for negotiation to complete
	wait(t, ctx, signaller2.NegotiationDone())
}
