package routes_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/peer-calls/peer-calls/server/config"
	"github.com/peer-calls/peer-calls/server/factory/adapter"
	"github.com/peer-calls/peer-calls/server/room"
	"github.com/peer-calls/peer-calls/server/routes"
	"github.com/peer-calls/peer-calls/server/wrtc/signals"
	"github.com/peer-calls/peer-calls/server/wrtc/tracks"
	"github.com/peer-calls/peer-calls/server/ws"
	"github.com/peer-calls/peer-calls/server/ws/wsmessage"
	"github.com/peer-calls/peer-calls/server/wshandler"
	"github.com/pion/webrtc/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

type addedPeer struct {
	room           string
	clientID       string
	peerConnection tracks.PeerConnection
	closeChannel   chan struct{}
}

type mockTracksManager struct {
	added chan addedPeer
}

func newMockTracksManager() *mockTracksManager {
	return &mockTracksManager{
		added: make(chan addedPeer, 10),
	}
}

func (m *mockTracksManager) Add(room string, clientID string, peerConnection tracks.PeerConnection, dataChannel *webrtc.DataChannel, signaller tracks.Signaller) <-chan struct{} {
	closeChannel := make(chan struct{})
	m.added <- addedPeer{
		room:           room,
		clientID:       clientID,
		peerConnection: peerConnection,
		closeChannel:   closeChannel,
	}
	return closeChannel
}

func setupP2SServer(rooms routes.RoomManager) (server *httptest.Server, url string) {
	handler := routes.NewPeerToServerRoomHandler(
		wshandler.NewWSS(rooms),
		[]config.ICEServer{},
		config.NetworkConfigSFU{},
		tracks.NewTracksManager(),
	)
	server = httptest.NewServer(handler)
	url = "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/" + roomName + "/" + clientID
	return
}

func TestWS_P2S_ConnectDisconnect(t *testing.T) {
	newAdapter := adapter.NewAdapterFactory(config.StoreConfig{})
	defer newAdapter.Close()
	rooms := room.NewRoomManager(newAdapter.NewAdapter)
	server, url := setupP2SServer(rooms)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	wsc := mustDialWS(t, ctx, url)
	err := wsc.Close(websocket.StatusNormalClosure, "")
	require.Nil(t, err, "error closing client socket")
}

func TestWS_P2S_Peer(t *testing.T) {
	// TODO fix mediaEngine should not be touched
	mediaEngine := webrtc.MediaEngine{}
	newAdapter := adapter.NewAdapterFactory(config.StoreConfig{})
	defer newAdapter.Close()
	rooms := room.NewRoomManager(newAdapter.NewAdapter)
	server, url := setupP2SServer(rooms)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	wsc := mustDialWS(t, ctx, url)
	defer wsc.Close(websocket.StatusNormalClosure, "")

	wsClient := ws.NewClientWithID(wsc, clientID)
	msgChan := wsClient.Subscribe(ctx)

	err := wsClient.Write(wsmessage.NewMessage("ready", roomName, map[string]interface{}{
		"nickname": "some-user",
	}))
	require.NoError(t, err, "error sending ready message")

loop:
	for {
		select {
		case msg := <-msgChan:
			t.Log("1 msg.Type", msg)
			if msg.Type == "users" {
				assert.Equal(t, "users", msg.Type)
				assert.Equal(t, roomName, msg.Room)
				payload := msg.Payload.(map[string]interface{})
				assert.Equal(t, "__SERVER__", payload["initiator"])
				assert.Equal(t, []interface{}{"__SERVER__"}, payload["peerIds"])
				assert.Equal(t, map[string]interface{}{clientID: "some-user"}, payload["nicknames"])
				break loop
			}
		case <-ctx.Done():
			t.Errorf("context timeout: %s", ctx.Err())
			return
		}
	}
	require.Nil(t, wsClient.Err())

	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	require.Nil(t, err, "error creating peer connection")

	signaller, err := signals.NewSignaller(
		false,
		pc,
		&mediaEngine,
		clientID,
		"__SERVER__",
	)
	defer signaller.Close() // also closes pc
	signalChan := signaller.SignalChannel()

	go func() {
		t.Log("listening for events")
		for msgChan != nil && signalChan != nil {
			select {
			case msg, ok := <-msgChan:
				t.Log("ws message")
				if !ok {
					msgChan = nil
					continue
				}
				if msg.Type == "signal" {
					payload, ok := msg.Payload.(map[string]interface{})
					require.True(t, ok, "invalid signal msg payload type")
					err := signaller.Signal(payload)
					require.NoError(t, err, "error in receiving signal payload: %w", err)
				}
			case signal, ok := <-signalChan:
				t.Log("signal")
				if !ok {
					signalChan = nil
					continue
				}
				err := wsClient.Write(wsmessage.NewMessage("signal", roomName, signal))
				require.NoError(t, err, "error sending singal to ws: %w", err)
			}
		}
	}()

	waitCh := make(chan struct{})
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		t.Log("connection state", state)
	})
	pc.OnDataChannel(func(d *webrtc.DataChannel) {
		close(waitCh)
	})
	select {
	case <-waitCh:
		t.Log("Got data channel")
	case <-ctx.Done():
		t.Errorf("context timeout: %s", ctx.Err())
	}
}
