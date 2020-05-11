package server_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/peer-calls/peer-calls/server"
	"github.com/pion/webrtc/v2"
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
	url = "ws" + strings.TrimPrefix(s.URL, "http") + "/ws/" + roomName + "/" + clientID
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
				assert.Equal(t, map[string]interface{}{clientID: "some-user"}, payload["nicknames"])
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
	// t.Log("listening for events")
	for wsRecvCh != nil && signalChan != nil {
		select {
		case msg, ok := <-wsRecvCh:
			if !ok {
				wsRecvCh = nil
				continue
			}
			// t.Log("ws message", msg.Type, msg.Payload)
			if msg.Type == "signal" {
				payload, ok := msg.Payload.(map[string]interface{})
				require.True(t, ok, "invalid signal msg payload type")
				err := signaller.Signal(payload)
				require.NoError(t, err, "error in receiving signal payload: %w", err)
			}
		case signal, ok := <-signalChan:
			// t.Log("signal", signal)
			if !ok {
				signalChan = nil
				continue
			}
			err := wsClient.Write(server.NewMessage("signal", roomName, signal))
			require.NoError(t, err, "error sending singal to ws: %w", err)
		}
	}
}

func createPeerConnection(t *testing.T, ctx context.Context, url string, clientID string) (pc *webrtc.PeerConnection, cleanup func() error) {
	t.Helper()
	var cleanups []func() error
	cleanup = func() error {
		var err error
		for _, clean := range cleanups {
			err2 := clean()
			if err == nil {
				err = err2
			}
		}
		return err
	}
	// TODO fix mediaEngine should not be touched
	mediaEngine := webrtc.MediaEngine{}

	wsc := mustDialWS(t, ctx, url)
	cleanups = append(cleanups, func() error {
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

	signaller, err := server.NewSignaller(
		loggerFactory,
		false,
		pc,
		&mediaEngine,
		clientID,
		"__SERVER__",
	)
	require.Nil(t, err, "error creating signaller")

	cleanups = append(cleanups, signaller.Close) // also closes pc

	go startSignalling(t, wsClient, msgChan, signaller)

	return
}

func waitPeerConnected(t *testing.T, ctx context.Context, pc *webrtc.PeerConnection) {
	t.Helper()
	waitCh := make(chan struct{})
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateConnected {
			pc.OnConnectionStateChange(nil)
			close(waitCh)
		}
	})
	select {
	case <-waitCh:
		// connected
	case <-ctx.Done():
		t.Errorf("context timeout: %s", ctx.Err())
	}
}

func TestSFU_PeerConnection(t *testing.T) {
	defer goleak.VerifyNone(t)
	newAdapter := server.NewAdapterFactory(loggerFactory, server.StoreConfig{})
	defer newAdapter.Close()
	rooms := server.NewAdapterRoomManager(newAdapter.NewAdapter)
	srv, url := setupSFUServer(rooms)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	pc, cleanup := createPeerConnection(t, ctx, url, clientID)
	defer cleanup()
	waitPeerConnected(t, ctx, pc)
}
