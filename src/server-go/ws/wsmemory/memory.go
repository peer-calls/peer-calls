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
func (m *MemoryAdapter) Broadcast(msg []byte) {
	m.clientsMu.RLock()
	m.broadcast(msg)
	m.clientsMu.RUnlock()
}

func (m *MemoryAdapter) broadcast(msg []byte) {
	for clientID := range m.clients {
		m.emit(clientID, msg)
	}
}

// Sends a message to specific socket.
func (m *MemoryAdapter) Emit(clientID string, msg []byte) {
	m.clientsMu.RLock()
	m.emit(clientID, msg)
	m.clientsMu.RUnlock()
}

func (m *MemoryAdapter) emit(clientID string, msg []byte) {
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
