package routes_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jeremija/peer-calls/src/server-go/routes"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsadapter"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsmessage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

type MockRoomManager struct {
	enter chan string
	exit  chan string
	emit  chan Emit
}

type Emit struct {
	clientID string
	message  wsmessage.Message
}

func NewMockRoomManager() *MockRoomManager {
	return &MockRoomManager{
		enter: make(chan string, 10),
		exit:  make(chan string, 10),
		emit:  make(chan Emit, 10),
	}
}

func (r *MockRoomManager) Enter(room string) wsadapter.Adapter {
	r.enter <- room
	return &MockAdapter{room: room, emit: r.emit}
}

func (r *MockRoomManager) Exit(room string) {
	r.exit <- room
}

func (r *MockRoomManager) close() {
	close(r.enter)
	close(r.exit)
	close(r.emit)
}

type MockAdapter struct {
	room string
	emit chan Emit
}

func (m *MockAdapter) Add(client wsadapter.Client) error {
	return nil
}

func (m *MockAdapter) Remove(clientID string) error {
	return nil
}

func (m *MockAdapter) Broadcast(message wsmessage.Message) error {
	return nil
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

func (m *MockAdapter) Emit(clientID string, message wsmessage.Message) error {
	m.emit <- Emit{
		clientID: clientID,
		message:  message,
	}
	return nil
}

const roomName = "test-room"
const clientID = "user1234"

func mustDialWS(t *testing.T, ctx context.Context, url string) *websocket.Conn {
	t.Helper()
	ws, _, err := websocket.Dial(ctx, url, nil)
	require.Nil(t, err)
	return ws
}

var serializer wsmessage.ByteSerializer

func mustWriteWS(t *testing.T, ctx context.Context, ws *websocket.Conn, msg wsmessage.Message) {
	t.Helper()
	data, err := serializer.Serialize(msg)
	require.Nil(t, err, "Error serializing message")
	err = ws.Write(ctx, websocket.MessageText, data)
	require.Nil(t, err, "Error writing message")
}

func mustReadWS(t *testing.T, ctx context.Context, ws *websocket.Conn) wsmessage.Message {
	t.Helper()
	messageType, data, err := ws.Read(ctx)
	require.Equal(t, websocket.MessageText, messageType, "Expected to read text message")
	msg, err := serializer.Deserialize(data)
	require.Nil(t, err, "Error deserializing message")
	return msg
}

func setupServer(rooms routes.RoomManager) (server *httptest.Server, url string) {
	wss := routes.NewWSS(rooms)
	server = httptest.NewServer(wss)
	url = "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/" + roomName + "/" + clientID
	return
}

func TestWS_disconnect(t *testing.T) {
	rooms := NewMockRoomManager()
	defer rooms.close()
	server, url := setupServer(rooms)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
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

func TestWS_event_ready(t *testing.T) {
	rooms := NewMockRoomManager()
	defer rooms.close()
	server, url := setupServer(rooms)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	ws := mustDialWS(t, ctx, url)
	mustWriteWS(t, ctx, ws, wsmessage.NewMessage("ready", "test-room", nil))
	msg := mustReadWS(t, ctx, ws)
	assert.Equal(t, "users", msg.Type)
	payload, ok := msg.Payload.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, map[string]interface{}{"client1": "abc"}, payload)
}

func TestWS_event_signal(t *testing.T) {
	rooms := NewMockRoomManager()
	defer rooms.close()
	server, url := setupServer(rooms)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	ws := mustDialWS(t, ctx, url)
	otherClientID := "other-user"
	signal := "a-signal"
	mustWriteWS(t, ctx, ws, wsmessage.NewMessage("signal", "test-room", map[string]string{
		"userId": otherClientID,
		"signal": signal,
	}))
	emit, ok := <-rooms.emit
	require.True(t, ok, "rooms.emit channel is closed")
	assert.Equal(t, emit.clientID, otherClientID)
	assert.Equal(t, "signal", emit.message.Type)
	payload, ok := emit.message.Payload.(map[string]string)
	require.True(t, ok, "unexpected payload type: %s", emit.message.Payload)
	assert.Equal(t, signal, payload["signal"])
	assert.Equal(t, clientID, payload["userId"])
}
