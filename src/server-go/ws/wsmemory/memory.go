package wsmemory

import (
	"fmt"
	"sync"

	"github.com/jeremija/peer-calls/src/server-go/ws/wsmessage"
)

type Client interface {
	ID() string
	WriteChannel() chan<- wsmessage.Message
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
func (m *MemoryAdapter) Add(client Client) (err error) {
	m.clientsMu.Lock()
	clientID := client.ID()
	m.clients[clientID] = client
	err = m.broadcast(wsmessage.NewMessageRoomJoin(m.room, clientID))
	m.clientsMu.Unlock()
	return
}

// Remove a client from the room
func (m *MemoryAdapter) Remove(clientID string) (err error) {
	m.clientsMu.Lock()
	err = m.broadcast(wsmessage.NewMessageRoomLeave(m.room, clientID))
	delete(m.clients, clientID)
	m.clientsMu.Unlock()
	return
}

func (m *MemoryAdapter) Clients() (clientIDs []string, err error) {
	m.clientsMu.RLock()
	for clientID := range m.clients {
		clientIDs = append(clientIDs, clientID)
	}
	m.clientsMu.RUnlock()
	return
}

func (m *MemoryAdapter) Size() (value int, err error) {
	m.clientsMu.RLock()
	value = len(m.clients)
	m.clientsMu.RUnlock()
	return
}

// Send a message to all sockets
func (m *MemoryAdapter) Broadcast(msg wsmessage.Message) error {
	m.clientsMu.RLock()
	err := m.broadcast(msg)
	m.clientsMu.RUnlock()
	return err
}

func (m *MemoryAdapter) broadcast(msg wsmessage.Message) (err error) {
	for clientID := range m.clients {
		if emitErr := m.emit(clientID, msg); emitErr != nil && err == nil {
			err = emitErr
		}
	}
	return
}

// Sends a message to specific socket.
func (m *MemoryAdapter) Emit(clientID string, msg wsmessage.Message) error {
	m.clientsMu.RLock()
	m.emit(clientID, msg)
	m.clientsMu.RUnlock()
	return nil
}

func (m *MemoryAdapter) emit(clientID string, msg wsmessage.Message) error {
	client, ok := m.clients[clientID]
	if !ok {
		return fmt.Errorf("Client not found, clientID: %s", clientID)
	}
	select {
	case client.WriteChannel() <- msg:
		return nil
	default:
		return fmt.Errorf("Client buffer full, clientID: %s", clientID)
		// if the client buffer is full, it will not be sent
	}
}
