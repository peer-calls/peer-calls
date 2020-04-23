package room

import (
	"sync"

	"github.com/peer-calls/peer-calls/server/ws/wsadapter"
)

type AdapterFactory func(room string) wsadapter.Adapter

type adapterCounter struct {
	count   uint64
	adapter wsadapter.Adapter
}

type RoomManager struct {
	rooms      map[string]*adapterCounter
	roomsMu    sync.RWMutex
	newAdapter AdapterFactory
}

func NewRoomManager(newAdapter AdapterFactory) *RoomManager {
	return &RoomManager{
		rooms:      map[string]*adapterCounter{},
		newAdapter: newAdapter,
	}
}

func (r *RoomManager) Enter(room string) wsadapter.Adapter {
	r.roomsMu.Lock()
	adapter, ok := r.rooms[room]
	if ok {
		adapter.count++
	} else {
		adapter = &adapterCounter{
			count:   1,
			adapter: r.newAdapter(room),
		}
		r.rooms[room] = adapter
	}
	r.roomsMu.Unlock()
	return adapter.adapter
}

func (r *RoomManager) Exit(room string) {
	r.roomsMu.Lock()
	adapter, ok := r.rooms[room]
	if ok {
		adapter.count--
		if adapter.count == 0 {
			delete(r.rooms, room)
			adapter.adapter.Close() // FIXME log error
		}
	}
	r.roomsMu.Unlock()
}
