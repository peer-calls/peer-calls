package server

import (
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/message"
)

type MemoryAdapter struct {
	clientsMu *sync.RWMutex
	clients   map[identifiers.ClientID]ClientWriter
	room      identifiers.RoomID
}

func NewMemoryAdapter(room identifiers.RoomID) *MemoryAdapter {
	var clientsMu sync.RWMutex

	return &MemoryAdapter{
		clientsMu: &clientsMu,
		clients:   map[identifiers.ClientID]ClientWriter{},
		room:      room,
	}
}

// Add a client to the room. Will return an error on duplicate client ID.
func (m *MemoryAdapter) Add(client ClientWriter) (err error) {
	m.clientsMu.Lock()

	clientID := client.ID()

	if _, ok := m.clients[clientID]; ok {
		err = errors.Annotatef(ErrDuplicateClientID, "%s", clientID)
	} else {
		m.clients[clientID] = client
	}

	m.clientsMu.Unlock()

	if err != nil {
		return errors.Trace(err)
	}

	err = m.broadcast(
		message.NewRoomJoin(m.room, message.RoomJoin{
			ClientID: clientID,
			Metadata: client.Metadata(),
		}),
	)
	return errors.Annotatef(err, "add client: %s", clientID)
}

func (m *MemoryAdapter) Close() error {
	return nil
}

// Remove a client from the room
func (m *MemoryAdapter) Remove(clientID identifiers.ClientID) (err error) {
	m.clientsMu.Lock()
	delete(m.clients, clientID)
	err = m.broadcast(message.NewRoomLeave(m.room, clientID))
	m.clientsMu.Unlock()
	return errors.Annotatef(err, "remove client: %s", clientID)
}

func (m *MemoryAdapter) Metadata(clientID identifiers.ClientID) (metadata string, ok bool) {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()
	client, ok := m.clients[clientID]

	if ok {
		metadata = client.Metadata()
	}

	return
}

func (m *MemoryAdapter) SetMetadata(clientID identifiers.ClientID, metadata string) (ok bool) {
	m.clientsMu.Lock()
	defer m.clientsMu.Unlock()

	client, ok := m.clients[clientID]
	if ok {
		client.SetMetadata(metadata)
	}

	return ok
}

// Returns clients with metadata
func (m *MemoryAdapter) Clients() (clientIDs map[identifiers.ClientID]string, err error) {
	m.clientsMu.RLock()
	clientIDs = make(map[identifiers.ClientID]string, len(m.clients))

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
func (m *MemoryAdapter) Broadcast(msg message.Message) error {
	m.clientsMu.RLock()
	err := m.broadcast(msg)
	m.clientsMu.RUnlock()
	return errors.Annotate(err, "Broadcast")
}

func (m *MemoryAdapter) broadcast(msg message.Message) error {
	var errs MultiErrorHandler

	for clientID := range m.clients {
		if err := m.emit(clientID, msg); err == nil {
			errs.Add(errors.Annotatef(err, "broadcast"))
		}
	}

	return errors.Trace(errs.Err())
}

// Sends a message to specific socket.
func (m *MemoryAdapter) Emit(clientID identifiers.ClientID, msg message.Message) error {
	m.clientsMu.RLock()
	err := m.emit(clientID, msg)
	m.clientsMu.RUnlock()
	return errors.Annotatef(err, "emit")
}

func (m *MemoryAdapter) emit(clientID identifiers.ClientID, msg message.Message) error {
	client, ok := m.clients[clientID]
	if !ok {
		return errors.Errorf("Client not found, clientID: %s", clientID)
	}

	err := client.Write(msg)

	return errors.Annotatef(err, "emit, clientID: %s", clientID)
}
