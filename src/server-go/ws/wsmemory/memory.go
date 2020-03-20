package wsmemory

import (
	"sync"
)

type Client interface {
	ID() string
	Messages() chan<- []byte
}

type MemoryAdapter struct {
	clientsMu *sync.RWMutex
	clients   map[string]Client
}

func NewMemoryAdapter() *MemoryAdapter {
	var clientsMu sync.RWMutex
	return &MemoryAdapter{
		clientsMu: &clientsMu,
		clients:   map[string]Client{},
	}
}

// Add a client to the room
func (m *MemoryAdapter) Add(client Client) {
	m.clientsMu.Lock()
	m.clients[client.ID()] = client
	m.clientsMu.Unlock()
}

// Remove a client from the room
func (m *MemoryAdapter) Remove(clientID string) {
	m.clientsMu.Lock()
	delete(m.clients, clientID)
	m.clientsMu.Unlock()
}

func (m *MemoryAdapter) Size() int {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()
	return len(m.clients)
}

// Send a message to all sockets
func (m *MemoryAdapter) Broadcast(msg []byte) {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()

	for clientID := range m.clients {
		m.Emit(clientID, msg)
	}
}

// Sends a message to specific socket.
func (m *MemoryAdapter) Emit(clientID string, msg []byte) {
	client, ok := m.clients[clientID]
	if !ok {
		return
	}
	select {
	case client.Messages() <- msg:
	default:
		// TODO see if this is called when channel is closed
	}
}
