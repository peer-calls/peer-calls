package server_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/peer-calls/peer-calls/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"nhooyr.io/websocket"
)

const timeout = 10 * time.Second

type MockRoomManager struct {
	enter     chan string
	exit      chan string
	emit      chan Emit
	broadcast chan server.Message
}

type Emit struct {
	clientID string
	message  server.Message
}

func NewMockRoomManager() *MockRoomManager {
	return &MockRoomManager{
		enter:     make(chan string, 10),
		exit:      make(chan string, 10),
		emit:      make(chan Emit, 10),
		broadcast: make(chan server.Message, 10),
	}
}

func (r *MockRoomManager) Enter(room string) server.Adapter {
	r.enter <- room
	return &MockAdapter{room: room, emit: r.emit, broadcast: r.broadcast}
}

func (r *MockRoomManager) Exit(room string) {
	r.exit <- room
}

func (r *MockRoomManager) close() {
	close(r.enter)
	close(r.exit)
	close(r.emit)
	close(r.broadcast)
}

type MockAdapter struct {
	room      string
	emit      chan Emit
	broadcast chan server.Message
}

func (m *MockAdapter) Add(client server.ClientWriter) error {
	return nil
}

func (m *MockAdapter) Remove(clientID string) error {
	return nil
}

func (m *MockAdapter) Broadcast(message server.Message) error {
	m.broadcast <- message
	return nil
}

func (m *MockAdapter) SetMetadata(clientID string, metadata string) bool {
	return true
}

func (m *MockAdapter) Clients() (map[string]string, error) {
	return map[string]string{"client1": "abc"}, nil
}

func (m *MockAdapter) Close() error {
	return nil
}

func (m *MockAdapter) Size() (int, error) {
	return 0, nil
}

func (m *MockAdapter) Metadata(clientID string) (string, bool) {
	return "", true
}

func (m *MockAdapter) Emit(clientID string, message server.Message) error {
	m.emit <- Emit{
		clientID: clientID,
		message:  message,
	}
	return nil
}

const roomName = "test-room"
const clientID = "user1"
const clientID2 = "user2"

func mustDialWS(t *testing.T, ctx context.Context, url string) *websocket.Conn {
	t.Helper()
	ws, _, err := websocket.Dial(ctx, url, nil)
	require.Nil(t, err)
	return ws
}

func mustWriteWS(t *testing.T, ctx context.Context, ws *websocket.Conn, msg server.Message) {
	t.Helper()
	data, err := serializer.Serialize(msg)
	require.Nil(t, err, "Error serializing message")
	err = ws.Write(ctx, websocket.MessageText, data)
	require.Nil(t, err, "Error writing message")
}

func mustReadWS(t *testing.T, ctx context.Context, ws *websocket.Conn) server.Message {
	t.Helper()
	messageType, data, err := ws.Read(ctx)
	require.NoError(t, err, "Error reading text message")
	require.Equal(t, websocket.MessageText, messageType, "Expected to read text message")
	msg, err := serializer.Deserialize(data)
	require.Nil(t, err, "Error deserializing message")
	return msg
}

func setupMeshServer(rooms server.RoomManager) (s *httptest.Server, url string) {
	handler := server.NewMeshHandler(loggerFactory, server.NewWSS(loggerFactory, rooms))
	s = httptest.NewServer(handler)
	url = "ws" + strings.TrimPrefix(s.URL, "http") + "/ws/" + roomName + "/" + clientID
	return
}

func TestMesh_disconnect(t *testing.T) {
	defer goleak.VerifyNone(t)
	rooms := NewMockRoomManager()
	defer rooms.close()
	server, url := setupMeshServer(rooms)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ws := mustDialWS(t, ctx, url)
	err := ws.Close(websocket.StatusNormalClosure, "")
	require.Nil(t, err, "error closing client socket")
	room, ok := <-rooms.enter
	assert.True(t, ok, "cannot read rooms enter vent")
	assert.Equal(t, roomName, room)
	room, ok = <-rooms.exit
	assert.True(t, ok, "cannot read rooms exit event")
	assert.Equal(t, roomName, room)
}

func TestMesh_event_ready(t *testing.T) {
	defer goleak.VerifyNone(t)
	rooms := NewMockRoomManager()
	defer rooms.close()
	srv, url := setupMeshServer(rooms)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ws := mustDialWS(t, ctx, url)
	defer func() { <-rooms.exit }()
	defer ws.Close(websocket.StatusGoingAway, "")
	mustWriteWS(t, ctx, ws, server.NewMessage("ready", "test-room", map[string]interface{}{
		"nickname": "abc",
	}))
	// msg := mustReadWS(t, ctx, ws)
	msg := <-rooms.broadcast
	assert.Equal(t, "users", msg.Type)
	payload, ok := msg.Payload.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, map[string]interface{}{
		"initiator": clientID,
		"peerIds":   []string{"client1"},
		"nicknames": map[string]string{
			"client1": "abc",
		},
	}, payload)
}

func TestMesh_event_signal(t *testing.T) {
	defer goleak.VerifyNone(t)
	rooms := NewMockRoomManager()
	defer rooms.close()
	srv, url := setupMeshServer(rooms)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ws := mustDialWS(t, ctx, url)
	defer func() { <-rooms.exit }()
	defer ws.Close(websocket.StatusGoingAway, "")
	otherClientID := "other-user"
	var signal interface{} = "a-signal"
	mustWriteWS(t, ctx, ws, server.NewMessage("signal", "test-room", map[string]interface{}{
		"userId": otherClientID,
		"signal": signal,
	}))
	emit, ok := <-rooms.emit
	require.True(t, ok, "rooms.emit channel is closed")
	assert.Equal(t, emit.clientID, otherClientID)
	assert.Equal(t, "signal", emit.message.Type)
	payload, ok := emit.message.Payload.(map[string]interface{})
	require.True(t, ok, "unexpected payload type: %s", emit.message.Payload)
	assert.Equal(t, signal, payload["signal"])
	assert.Equal(t, clientID, payload["userId"])
}
