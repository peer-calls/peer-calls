package server

import (
	"io"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
)

type NewAdapterFunc func(room identifiers.RoomID) Adapter

type adapterCounter struct {
	count   uint64
	adapter Adapter
}

type AdapterRoomManager struct {
	rooms      map[identifiers.RoomID]*adapterCounter
	roomsMu    sync.RWMutex
	newAdapter NewAdapterFunc
}

var _ RoomManager = &AdapterRoomManager{}

func NewAdapterRoomManager(newAdapter NewAdapterFunc) *AdapterRoomManager {
	return &AdapterRoomManager{
		rooms:      map[identifiers.RoomID]*adapterCounter{},
		newAdapter: newAdapter,
	}
}

func (r *AdapterRoomManager) Enter(room identifiers.RoomID) (adapter Adapter, isNew bool) {
	r.roomsMu.Lock()
	defer r.roomsMu.Unlock()

	ac, ok := r.rooms[room]
	if ok {
		ac.count++
	} else {
		isNew = true
		ac = &adapterCounter{
			count:   1,
			adapter: r.newAdapter(room),
		}
		r.rooms[room] = ac
	}
	return ac.adapter, isNew
}

func (r *AdapterRoomManager) Exit(room identifiers.RoomID) (isRemoved bool) {
	r.roomsMu.Lock()
	defer r.roomsMu.Unlock()

	adapter, ok := r.rooms[room]
	if ok {
		adapter.count--
		if adapter.count == 0 {
			isRemoved = true

			delete(r.rooms, room)

			adapter.adapter.Close() // FIXME log error
		}
	}

	return isRemoved
}

type ChannelRoomManager struct {
	roomManager         RoomManager
	roomEventsChan      chan RoomEvent
	closedChan          chan struct{}
	closedChanCloseOnce sync.Once
	mu                  sync.Mutex
}

func NewChannelRoomManager(roomManager RoomManager) *ChannelRoomManager {
	return &ChannelRoomManager{
		roomManager:    roomManager,
		roomEventsChan: make(chan RoomEvent),
		closedChan:     make(chan struct{}),
	}
}

// Close exists for tests. This channel should always stay open IRL.
func (r *ChannelRoomManager) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.closedChanCloseOnce.Do(func() {
		close(r.roomEventsChan)
	})
}

func (r *ChannelRoomManager) isClosed() bool {
	select {
	case <-r.closedChan:
		return true
	default:
		return false
	}
}

func (r *ChannelRoomManager) Enter(room identifiers.RoomID) (adapter Adapter, isNew bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	adapter, isNew = r.roomManager.Enter(room)
	if isNew && !r.isClosed() {
		r.roomEventsChan <- RoomEvent{
			RoomName: room,
			Type:     RoomEventTypeAdd,
		}
	}
	return adapter, isNew
}

func (r *ChannelRoomManager) Exit(room identifiers.RoomID) (isRemoved bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	isRemoved = r.roomManager.Exit(room)
	if isRemoved && !r.isClosed() {
		r.roomEventsChan <- RoomEvent{
			RoomName: room,
			Type:     RoomEventTypeRemove,
		}
	}
	return isRemoved
}

func (r *ChannelRoomManager) AcceptEvent() (RoomEvent, error) {
	event, ok := <-r.roomEventsChan
	if !ok {
		return event, errors.Annotatef(io.ErrClosedPipe, "ChannelRoomManager closed")
	}

	return event, nil
}

func (r *ChannelRoomManager) RoomEventsChannel() <-chan RoomEvent {
	return r.roomEventsChan
}

type RoomEvent struct {
	RoomName identifiers.RoomID
	Type     RoomEventType
}

type RoomEventType int

const (
	RoomEventTypeAdd RoomEventType = iota + 1
	RoomEventTypeRemove
)
