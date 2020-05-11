package server_test

import (
	"context"
	"io"
	"math/rand"
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

func setupSFUServer(rooms server.RoomManager) (s *httptest.Server, url string) {
	handler := server.NewSFUHandler(
		loggerFactory,
		server.NewWSS(loggerFactory, rooms),
		[]server.ICEServer{},
		server.NetworkConfigSFU{},
		server.NewMemoryTracksManager(loggerFactory),
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
	server, url := setupSFUServer(rooms)
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
			// t.Log("ws message", msg.Type, msg.Payload)
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
			// t.Log("signal", signal)
			err := wsClient.Write(server.NewMessage("signal", roomName, signal))
			require.NoError(t, err, "error sending signal to ws: %w", err)
		}
	}()
}

func createPeerConnection(t *testing.T, ctx context.Context, url string, clientID string) (pc *webrtc.PeerConnection, signaller *server.Signaller, cleanup func() error) {
	t.Helper()
	var closer test.TestCloser
	cleanup = closer.Close
	// TODO fix mediaEngine should not be touched
	mediaEngine := webrtc.MediaEngine{}

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

	pc, err = webrtc.NewPeerConnection(webrtc.Configuration{})
	require.Nil(t, err, "error creating peer connection")

	signaller, err = server.NewSignaller(
		loggerFactory,
		false,
		pc,
		&mediaEngine,
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
	srv, wsBaseURL := setupSFUServer(rooms)
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
	defer newAdapter.Close()
	rooms := server.NewAdapterRoomManager(newAdapter.NewAdapter)
	srv, wsBaseURL := setupSFUServer(rooms)
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
	pc2.OnTrack(func(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
		t.Log("OnTrack", track.SSRC())
		onTrackFiredDone()
		for {
			_, err := track.ReadRTP()
			if err == io.EOF {
				onTrackEOFDone()
				return
			} else if err != nil {
				t.Errorf("Error reading track: %s", err)
				return
			}
		}
	})

	track, err := pc1.NewTrack(webrtc.DefaultPayloadTypeVP8, rand.Uint32(), "track-one", "stream-one")
	require.NoError(t, err)

	sender, err := pc1.AddTrack(track)
	require.NoError(t, err)

	wait(t, ctx, signaller1.Negotiate())
	t.Log("Negotiate (1) done ======================================================")

	t.Log("sending video")
	sendVideoUntilDone(t, onTrackFired.Done(), track)
	t.Log("sending video done")
	assert.Equal(t, context.Canceled, onTrackFired.Err(), "test timed out")

	t.Log("stopping sender")

	// trigger io.EOF when reading from track2
	assert.NoError(t, pc1.RemoveTrack(sender))
	wait(t, ctx, signaller1.Negotiate())
	t.Log("Negotiate (2) done ======================================================")

	wait(t, ctx, onTrackEOF.Done())
	assert.Equal(t, context.Canceled, onTrackEOF.Err(), "test timed out")

	// server will want to negotiate after track is removed so we wait for negotiation to complete
	wait(t, ctx, signaller2.NegotiationDone())
	t.Log("NegotiationDone (2) done ======================================================")

	t.Log("-- test end --")
}
