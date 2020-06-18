package server

import (
	"sync"
)

type NewAdapterFunc func(room string) Adapter

type adapterCounter struct {
	count   uint64
	adapter Adapter
}

type AdapterRoomManager struct {
	rooms      map[string]*adapterCounter
	roomsMu    sync.RWMutex
	newAdapter NewAdapterFunc
}

var _ RoomManager = &AdapterRoomManager{}

func NewAdapterRoomManager(newAdapter NewAdapterFunc) *AdapterRoomManager {
	return &AdapterRoomManager{
		rooms:      map[string]*adapterCounter{},
		newAdapter: newAdapter,
	}
}

func (r *AdapterRoomManager) Enter(room string) (adapter Adapter, isNew bool) {
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

func (r *AdapterRoomManager) Exit(room string) (isRemoved bool) {
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
	roomManager    RoomManager
	roomEventsChan chan RoomEvent
}

func NewChannelRoomManager(roomManager RoomManager) *ChannelRoomManager {
	return &ChannelRoomManager{
		roomManager:    roomManager,
		roomEventsChan: make(chan RoomEvent),
	}
}

// Close exists for tests. This channel should always stay open IRL.
func (r *ChannelRoomManager) Close() {
	close(r.roomEventsChan)
}

func (r *ChannelRoomManager) Enter(room string) (adapter Adapter, isNew bool) {
	adapter, isNew = r.roomManager.Enter(room)
	if isNew {
		r.roomEventsChan <- RoomEvent{room, RoomEventTypeAdd}
	}
	return adapter, isNew
}

func (r *ChannelRoomManager) Exit(room string) (isRemoved bool) {
	isRemoved = r.roomManager.Exit(room)
	if isRemoved {
		r.roomEventsChan <- RoomEvent{room, RoomEventTypeRemove}
	}
	return isRemoved
}

func (r *ChannelRoomManager) RoomEventsChannel() <-chan RoomEvent {
	return r.roomEventsChan
}

type RoomEvent struct {
	RoomName string
	Type     RoomEventType
}

type RoomEventType int

const (
	RoomEventTypeAdd RoomEventType = iota + 1
	RoomEventTypeRemove
)
