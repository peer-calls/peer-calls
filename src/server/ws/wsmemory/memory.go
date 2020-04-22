package wsmemory

import (
	"fmt"
	"sync"

	"github.com/jeremija/peer-calls/src/server/ws/wsadapter"
	"github.com/jeremija/peer-calls/src/server/ws/wsmessage"
)

type MemoryAdapter struct {
	clientsMu *sync.RWMutex
	clients   map[string]wsadapter.Client
	room      string
}

func NewMemoryAdapter(room string) *MemoryAdapter {
	var clientsMu sync.RWMutex
	return &MemoryAdapter{
		clientsMu: &clientsMu,
		clients:   map[string]wsadapter.Client{},
		room:      room,
	}
}

// Add a client to the room
func (m *MemoryAdapter) Add(client wsadapter.Client) (err error) {
	m.clientsMu.Lock()
	clientID := client.ID()
	m.clients[clientID] = client
	err = m.broadcast(wsmessage.NewMessageRoomJoin(m.room, clientID, client.Metadata()))
	m.clientsMu.Unlock()
	return
}

func (m *MemoryAdapter) Close() error {
	return nil
}

// Remove a client from the room
func (m *MemoryAdapter) Remove(clientID string) (err error) {
	m.clientsMu.Lock()
	err = m.broadcast(wsmessage.NewMessageRoomLeave(m.room, clientID))
	delete(m.clients, clientID)
	m.clientsMu.Unlock()
	return
}

func (m *MemoryAdapter) Metadata(clientID string) (metadata string, ok bool) {
	client, ok := m.clients[clientID]
	if ok {
		metadata = client.Metadata()
	}
	return
}

func (m *MemoryAdapter) SetMetadata(clientID string, metadata string) (ok bool) {
	client, ok := m.clients[clientID]
	if ok {
		client.SetMetadata(metadata)
	}
	return ok
}

// Returns clients with metadata
func (m *MemoryAdapter) Clients() (clientIDs map[string]string, err error) {
	m.clientsMu.RLock()
	clientIDs = map[string]string{}
	for clientID, client := range m.clients {
		clientIDs[clientID] = client.Metadata()
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
		return fmt.Errorf("wsadapter.Client not found, clientID: %s", clientID)
	}
	err := client.Write(msg)
	if err != nil {
		return fmt.Errorf("MemoryAdapter.emit error: %w", err)
	}
	return nil
}
