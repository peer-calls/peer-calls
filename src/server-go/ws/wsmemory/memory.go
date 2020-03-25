package wsmemory

import (
	"sync"

	"github.com/jeremija/peer-calls/src/server-go/ws/wsmessage"
)

type Client interface {
	ID() string
	Messages() chan<- wsmessage.Message
}

type MemoryAdapter struct {
	clientsMu *sync.RWMutex
	clients   map[string]Client
	room      string
}

func NewMemoryAdapter(room string) *MemoryAdapter {
	var clientsMu sync.RWMutex
	return &MemoryAdapter{
		clientsMu: &clientsMu,
		clients:   map[string]Client{},
		room:      room,
	}
}

// Add a client to the room
func (m *MemoryAdapter) Add(client Client) {
	m.clientsMu.Lock()
	clientID := client.ID()
	m.clients[clientID] = client
	m.broadcast(wsmessage.NewMessageRoomJoin(m.room, clientID))
	m.clientsMu.Unlock()
}

// Remove a client from the room
func (m *MemoryAdapter) Remove(clientID string) {
	m.clientsMu.Lock()
	m.broadcast(wsmessage.NewMessageRoomLeave(m.room, clientID))
	delete(m.clients, clientID)
	m.clientsMu.Unlock()
}

func (m *MemoryAdapter) Clients() []string {
	m.clientsMu.RLock()
	clientIDs := []string{}
	for clientID := range m.clients {
		clientIDs = append(clientIDs, clientID)
	}
	m.clientsMu.RUnlock()
	return clientIDs
}

func (m *MemoryAdapter) Size() (value int) {
	m.clientsMu.RLock()
	value = len(m.clients)
	m.clientsMu.RUnlock()
	return
}

// Send a message to all sockets
func (m *MemoryAdapter) Broadcast(msg wsmessage.Message) {
	m.clientsMu.RLock()
	m.broadcast(msg)
	m.clientsMu.RUnlock()
}

func (m *MemoryAdapter) broadcast(msg wsmessage.Message) {
	for clientID := range m.clients {
		m.emit(clientID, msg)
	}
}

// Sends a message to specific socket.
func (m *MemoryAdapter) Emit(clientID string, msg wsmessage.Message) {
	m.clientsMu.RLock()
	m.emit(clientID, msg)
	m.clientsMu.RUnlock()
}

func (m *MemoryAdapter) emit(clientID string, msg wsmessage.Message) {
	client, ok := m.clients[clientID]
	if !ok {
		return
	}
	select {
	case client.Messages() <- msg:
	default:
		// if the client buffer is full, it will not be sent
	}
}
