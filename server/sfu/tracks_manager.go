package sfu

import (
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/pubsub"
	"github.com/peer-calls/peer-calls/server/transport"
)

const DataChannelName = "data"

type TracksManager struct {
	log                 logger.Logger
	mu                  sync.RWMutex
	peerManagers        map[string]*PeerManager
	jitterBufferEnabled bool
}

func NewTracksManager(log logger.Logger, jitterBufferEnabled bool) *TracksManager {
	return &TracksManager{
		log:                 log.WithNamespaceAppended("tracks_manager"),
		peerManagers:        map[string]*PeerManager{},
		jitterBufferEnabled: jitterBufferEnabled,
	}
}

func (m *TracksManager) Add(room string, transport transport.Transport) (<-chan pubsub.PubTrackEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	log := m.log.WithCtx(logger.Ctx{
		"room_id": room,
	})

	peerManager, ok := m.peerManagers[room]
	if !ok {
		jitterHandler := NewJitterHandler(
			log,
			m.jitterBufferEnabled,
		)
		peerManager = NewPeerManager(room, m.log, jitterHandler)
		m.peerManagers[room] = peerManager

		// TODO Write to RoomEventsChan
	}

	log = log.WithCtx(logger.Ctx{
		"client_id": transport.ClientID(),
	})

	log.Info("Add peer", nil)

	pubTrackEventsCh, err := peerManager.Add(transport)
	if err != nil {
		return nil, errors.Annotatef(err, "add transport")
	}

	go func() {
		<-transport.CloseChannel()
		m.mu.Lock()
		defer m.mu.Unlock()

		peerManager.Remove(transport.ClientID())

		// TODO tell the difference between server and webrtc transports since
		// server transports should not be counted, and they should be removed.
		if peerManager.Size() == 0 {
			peerManager.Close()
			// TODO write to RoomEventsChan
			delete(m.peerManagers, room)
		}
	}()

	return pubTrackEventsCh, nil
}

func (m *TracksManager) TracksMetadata(room string, clientID string) (metadata []TrackMetadata, ok bool) {
	m.mu.RLock()
	peerManager, ok := m.peerManagers[room]
	m.mu.RUnlock()

	if !ok {
		return metadata, false
	}

	return peerManager.TracksMetadata(clientID)
}

func (m *TracksManager) Sub(params SubParams) error {
	peerManager, ok := m.peerManagers[params.Room]
	if !ok {
		return errors.Errorf("room not found: %s", params.Room)
	}

	err := peerManager.Sub(params)

	return errors.Trace(err)
}

func (m *TracksManager) Unsub(params SubParams) error {
	peerManager, ok := m.peerManagers[params.Room]
	if !ok {
		return errors.Errorf("room not found: %s", params.Room)
	}

	err := peerManager.Unsub(params)

	return errors.Trace(err)
}
