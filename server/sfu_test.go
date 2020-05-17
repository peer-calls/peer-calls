package server_test

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/peer-calls/peer-calls/server"
	"github.com/peer-calls/peer-calls/server/test"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"nhooyr.io/websocket"
)

func setupSFUServer(rooms server.RoomManager, jitterBufferEnabled bool) (s *httptest.Server, url string) {
	handler := server.NewSFUHandler(
		loggerFactory,
		server.NewWSS(loggerFactory, rooms),
		[]server.ICEServer{},
		server.NetworkConfigSFU{},
		server.NewMemoryTracksManager(loggerFactory, jitterBufferEnabled),
	)
	s = httptest.NewServer(handler)
	url = "ws" + strings.TrimPrefix(s.URL, "http") + "/ws/"
	return
}

func TestSFU_ConnectDisconnect(t *testing.T) {
	defer goleak.VerifyNone(t)
	newAdapter := server.NewAdapterFactory(loggerFactory, server.StoreConfig{})
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
			if msg.Type == "users" {
				assert.Equal(t, "users", msg.Type)
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

func startSignalling(t *testing.T, wsClient *server.Client, wsRecvCh <-chan server.Message, signaller *server.Signaller) {
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
	var closer test.TestCloser
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

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(webrtc.SettingEngine{
			LoggerFactory: server.NewPionLoggerFactory(loggerFactory),
		}),
	)

	pc, err = api.NewPeerConnection(webrtc.Configuration{})
	require.Nil(t, err, "error creating peer connection")

	signaller, err = server.NewSignaller(
		loggerFactory,
		false,
		pc,
		clientID,
		"__SERVER__",
	)
	require.Nil(t, err, "error creating signaller")

	closer.AddFuncErr(signaller.Close) // also closes pc

	startSignalling(t, wsClient, msgChan, signaller)

	return
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
	defer goleak.VerifyNone(t)
	newAdapter := server.NewAdapterFactory(loggerFactory, server.StoreConfig{})
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
	defer goleak.VerifyNone(t)
	newAdapter := server.NewAdapterFactory(loggerFactory, server.StoreConfig{})
	log := loggerFactory.GetLogger("test")
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
		log.Println("OnTrack", remoteTrack.SSRC())
		onTrackFiredDone()
		for {
			_, err := remoteTrack.ReadRTP()
			if remoteTrack != nil {
				log.Printf("pc2 remote track ended")
				assert.Equal(t, io.EOF, err, "error reading track")
				onTrackEOFDone()
				return
			}
		}
	})

	localTrack, err := pc1.NewTrack(webrtc.DefaultPayloadTypeVP8, 12345, "track-one", "stream-one")
	require.NoError(t, err)

	log.Println("AddTrack start")
	sender, err := pc1.AddTrack(localTrack)
	log.Println("AddTrack end")
	require.NoError(t, err)
	require.NotNil(t, sender)

	go sendVideoUntilDone(t, onTrackFired.Done(), localTrack)

	log.Printf("sending negotiate request (1)")
	wait(t, ctx, signaller1.Negotiate())
	log.Println("Negotiate (1) done ======================================================")

	<-onTrackFired.Done()
	log.Println("sending video done")
	assert.Equal(t, context.Canceled, onTrackFired.Err(), "test timed out")

	log.Println("removing track")
	// trigger io.EOF when reading from track2
	assert.NoError(t, pc1.RemoveTrack(sender))

	log.Printf("sending negotiate request (2)")
	wait(t, ctx, signaller1.Negotiate())
	log.Println("Negotiate (2) done ======================================================")

	<-onTrackEOF.Done()
	assert.Equal(t, context.Canceled, onTrackEOF.Err(), "test timed out")

	log.Printf("waiting for peer2 negotiation to be done (3)")
	// server will want to negotiate after track is removed so we wait for negotiation to complete
	wait(t, ctx, signaller2.NegotiationDone())
	log.Println("Negotiation (3) done ======================================================")
}
