package server

import (
	"sync"

	"github.com/juju/errors"
)

type MemoryAdapter struct {
	clientsMu *sync.RWMutex
	clients   map[string]ClientWriter
	room      string
}

func NewMemoryAdapter(room string) *MemoryAdapter {
	var clientsMu sync.RWMutex
	return &MemoryAdapter{
		clientsMu: &clientsMu,
		clients:   map[string]ClientWriter{},
		room:      room,
	}
}

// Add a client to the room
func (m *MemoryAdapter) Add(client ClientWriter) (err error) {
	m.clientsMu.Lock()
	clientID := client.ID()
	m.clients[clientID] = client
	err = m.broadcast(NewMessageRoomJoin(m.room, clientID, client.Metadata()))
	m.clientsMu.Unlock()
	return errors.Annotatef(err, "add client: %s", clientID)
}

func (m *MemoryAdapter) Close() error {
	return nil
}

// Remove a client from the room
func (m *MemoryAdapter) Remove(clientID string) (err error) {
	m.clientsMu.Lock()
	delete(m.clients, clientID)
	err = m.broadcast(NewMessageRoomLeave(m.room, clientID))
	m.clientsMu.Unlock()
	return errors.Annotatef(err, "remove client: %s", clientID)
}

func (m *MemoryAdapter) Metadata(clientID string) (metadata string, ok bool) {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()
	client, ok := m.clients[clientID]

	if ok {
		metadata = client.Metadata()
	}

	return
}

func (m *MemoryAdapter) SetMetadata(clientID string, metadata string) (ok bool) {
	m.clientsMu.Lock()
	defer m.clientsMu.Unlock()

	client, ok := m.clients[clientID]
	if ok {
		client.SetMetadata(metadata)
	}

	return ok
}

// Returns clients with metadata
func (m *MemoryAdapter) Clients() (clientIDs map[string]string, err error) {
	m.clientsMu.RLock()
	clientIDs = make(map[string]string, len(m.clients))

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
func (m *MemoryAdapter) Broadcast(msg Message) error {
	m.clientsMu.RLock()
	err := m.broadcast(msg)
	m.clientsMu.RUnlock()
	return errors.Annotate(err, "Broadcast")
}

func (m *MemoryAdapter) broadcast(msg Message) error {
	var errs MultiErrorHandler

	for clientID := range m.clients {
		if err := m.emit(clientID, msg); err == nil {
			errs.Add(errors.Annotatef(err, "broadcast"))
		}
	}

	return errors.Trace(errs.Err())
}

// Sends a message to specific socket.
func (m *MemoryAdapter) Emit(clientID string, msg Message) error {
	m.clientsMu.RLock()
	err := m.emit(clientID, msg)
	m.clientsMu.RUnlock()
	return errors.Annotatef(err, "emit")
}

func (m *MemoryAdapter) emit(clientID string, msg Message) error {
	client, ok := m.clients[clientID]
	if !ok {
		return errors.Errorf("Client not found, clientID: %s", clientID)
	}

	err := client.Write(msg)

	return errors.Annotatef(err, "emit, clientID: %s", clientID)
}
