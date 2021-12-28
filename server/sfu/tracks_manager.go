package sfu

import (
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/pubsub"
	"github.com/peer-calls/peer-calls/v4/server/transport"
)

const DataChannelName = "data"

type TracksManager struct {
	log                 logger.Logger
	mu                  sync.RWMutex
	peerManagers        map[identifiers.RoomID]*PeerManager
	jitterBufferEnabled bool
}

func NewTracksManager(log logger.Logger, jitterBufferEnabled bool) *TracksManager {
	return &TracksManager{
		log:                 log.WithNamespaceAppended("tracks_manager"),
		peerManagers:        map[identifiers.RoomID]*PeerManager{},
		jitterBufferEnabled: jitterBufferEnabled,
	}
}

// Add adds a transport to the existing PeerManager. If the manager does not
// exist, it is created.
//
// NOTE: rooms are created when the peer joins the room over the WebSocket
// connection. The component in charge for this is the RoomManager.
//
// Add is called from two places:
//  - When WebRTCTransports are created and peers join the room, or
//  - When RoomManager event that a room was created: A server transport will
//    be created for each configured node.
func (m *TracksManager) Add(room identifiers.RoomID, tr transport.Transport) (<-chan pubsub.PubTrackEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	log := m.log.WithCtx(logger.Ctx{
		"room_id": room,
	})

	peerManager, ok := m.peerManagers[room]
	if !ok {
		log.Info("Add peer manager", nil)

		jitterHandler := NewJitterHandler(
			log,
			m.jitterBufferEnabled,
		)
		peerManager = NewPeerManager(room, log, jitterHandler)
		m.peerManagers[room] = peerManager
	}

	log = log.WithCtx(logger.Ctx{
		"client_id": tr.ClientID(),
	})

	log.Info("Add peer", nil)

	pubTrackEventsCh, err := peerManager.Add(tr)
	if err != nil {
		return nil, errors.Annotatef(err, "add transport")
	}

	go func() {
		<-tr.Done()
		m.mu.Lock()
		defer m.mu.Unlock()

		// Note: if this transport was already replaced in a previous call to Add,
		// Remove won't actually do anything - it will just return an error, but
		// there's no need to handle it.
		if err := peerManager.Remove(tr); err != nil {
			log.Error("Remove peer", errors.Trace(err), nil)
		} else {
			log.Info("Remove peer", nil)
		}

		// Since the server transports are created when room is created, and
		// removed when a room is removed, we don't need to do anything special
		// to count the number of non-server peers here.
		//
		// It is fine to check for the size because peerManager is only ever
		// modified from this component, and we are under a lock.
		if peerManager.Size() == 0 {
			log.Info("Remove peer manager", nil)

			peerManager.Close()

			delete(m.peerManagers, room)
		}
	}()

	return pubTrackEventsCh, nil
}

func (m *TracksManager) Sub(params SubParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	peerManager, ok := m.peerManagers[params.Room]
	if !ok {
		return errors.Errorf("room not found: %s", params.Room)
	}

	err := peerManager.Sub(params)

	return errors.Trace(err)
}

func (m *TracksManager) Unsub(params SubParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	peerManager, ok := m.peerManagers[params.Room]
	if !ok {
		return errors.Errorf("room not found: %s", params.Room)
	}

	err := peerManager.Unsub(params)

	return errors.Trace(err)
}
