package server_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"nhooyr.io/websocket"
)

const timeout = 10 * time.Second

type MockRoomManager struct {
	enter     chan identifiers.RoomID
	exit      chan identifiers.RoomID
	emit      chan Emit
	broadcast chan message.Message
}

type Emit struct {
	clientID identifiers.ClientID
	message  message.Message
}

func NewMockRoomManager() *MockRoomManager {
	return &MockRoomManager{
		enter:     make(chan identifiers.RoomID, 10),
		exit:      make(chan identifiers.RoomID, 10),
		emit:      make(chan Emit, 10),
		broadcast: make(chan message.Message, 10),
	}
}

var _ server.RoomManager = &MockRoomManager{}

func (r *MockRoomManager) Enter(room identifiers.RoomID) (server.Adapter, bool) {
	r.enter <- room

	return &MockAdapter{room: room, emit: r.emit, broadcast: r.broadcast}, true
}

func (r *MockRoomManager) Exit(room identifiers.RoomID) bool {
	r.exit <- room
	return false
}

func (r *MockRoomManager) close() {
	close(r.enter)
	close(r.exit)
	close(r.emit)
	close(r.broadcast)
}

type MockAdapter struct {
	room      identifiers.RoomID
	emit      chan Emit
	broadcast chan message.Message
}

func (m *MockAdapter) Add(client server.ClientWriter) error {
	return nil
}

func (m *MockAdapter) Remove(clientID identifiers.ClientID) error {
	return nil
}

func (m *MockAdapter) Broadcast(message message.Message) error {
	m.broadcast <- message
	return nil
}

func (m *MockAdapter) SetMetadata(clientID identifiers.ClientID, metadata string) bool {
	return true
}

func (m *MockAdapter) Clients() (map[identifiers.ClientID]string, error) {
	return map[identifiers.ClientID]string{"client1": "abc"}, nil
}

func (m *MockAdapter) Close() error {
	return nil
}

func (m *MockAdapter) Size() (int, error) {
	return 0, nil
}

func (m *MockAdapter) Metadata(clientID identifiers.ClientID) (string, bool) {
	return "", true
}

func (m *MockAdapter) Emit(clientID identifiers.ClientID, message message.Message) error {
	m.emit <- Emit{
		clientID: clientID,
		message:  message,
	}
	return nil
}

const roomName = identifiers.RoomID("test-room")
const clientID = identifiers.ClientID("user1")
const clientID2 = identifiers.ClientID("user2")

func mustDialWS(t *testing.T, ctx context.Context, url string) *websocket.Conn {
	t.Helper()
	ws, _, err := websocket.Dial(ctx, url, nil)
	require.Nil(t, err)
	return ws
}

func mustWriteWS(t *testing.T, ctx context.Context, ws *websocket.Conn, msg message.Message) {
	t.Helper()
	data, err := serializer.Serialize(msg)
	require.Nil(t, err, "Error serializing message")
	err = ws.Write(ctx, websocket.MessageText, data)
	require.Nil(t, err, "Error writing message")
}

func mustReadWS(t *testing.T, ctx context.Context, ws *websocket.Conn) message.Message {
	t.Helper()
	messageType, data, err := ws.Read(ctx)
	require.NoError(t, err, "Error reading text message")
	require.Equal(t, websocket.MessageText, messageType, "Expected to read text message")
	msg, err := serializer.Deserialize(data)
	require.Nil(t, err, "Error deserializing message")
	return msg
}

func setupMeshServer(rooms server.RoomManager) (s *httptest.Server, url string) {
	log := logger.New()
	handler := server.NewMeshHandler(log, server.NewWSS(log, rooms))
	s = httptest.NewServer(handler)
	url = "ws" + strings.TrimPrefix(s.URL, "http") + "/ws/" + roomName.String() + "/" + clientID.String()
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
	mustWriteWS(t, ctx, ws, message.NewReady("test-room", message.Ready{
		Nickname: "abc",
	}))
	// msg := mustReadWS(t, ctx, ws)
	msg := <-rooms.broadcast
	assert.Equal(t, message.TypeUsers, msg.Type)

	expUsers := message.Users{
		Initiator: clientID,
		PeerIDs:   []identifiers.ClientID{"client1"},
		Nicknames: map[identifiers.ClientID]string{
			"client1": "abc",
		},
	}

	require.Equal(t, expUsers, *msg.Payload.Users)
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
	otherClientID := identifiers.ClientID("other-user")

	signal := message.Signal{
		Type: message.SignalTypeOffer,
		SDP:  "-sdp-",
	}

	mustWriteWS(t, ctx, ws, message.NewSignal("test-room", message.UserSignal{
		PeerID: otherClientID,
		Signal: signal,
	}))
	emit, ok := <-rooms.emit
	require.True(t, ok, "rooms.emit channel is closed")
	assert.Equal(t, emit.clientID, otherClientID)
	assert.Equal(t, message.TypeSignal, emit.message.Type)

	require.NotNil(t, emit.message.Payload.Signal)
	assert.Equal(t, signal, emit.message.Payload.Signal.Signal)
	assert.Equal(t, clientID, emit.message.Payload.Signal.PeerID)
}
