package server_test

import (
	"sync"
	"testing"

	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestAdapterRoomManager(t *testing.T) {
	t.Parallel()

	var newAdapter server.NewAdapterFunc = func(room identifiers.RoomID) server.Adapter {
		return server.NewMemoryAdapter(room)
	}

	rooms := server.NewAdapterRoomManager(newAdapter)

	adapter1, isNew := rooms.Enter("test")
	assert.True(t, isNew)
	_, ok := adapter1.(*server.MemoryAdapter)
	require.True(t, ok)

	adapter2, isNew := rooms.Enter("test")
	assert.False(t, isNew)
	assert.True(t, adapter1 == adapter2, "adapters should be the same")

	isRemoved := rooms.Exit("test")
	assert.False(t, isRemoved)

	adapter3, isNew := rooms.Enter("test")
	assert.False(t, isNew)
	assert.True(t, adapter1 == adapter3, "adapters should be the same")

	isRemoved = rooms.Exit("test")
	assert.False(t, isRemoved)

	isRemoved = rooms.Exit("test")
	assert.True(t, isRemoved)

	adapter4, isNew := rooms.Enter("test")
	assert.True(t, isNew)
	assert.True(t, adapter1 != adapter4, "adapters should NOT be the same")
}

func TestChannelRoomManager(t *testing.T) {
	goleak.VerifyNone(t)
	defer goleak.VerifyNone(t)

	var newAdapter server.NewAdapterFunc = func(room identifiers.RoomID) server.Adapter {
		return server.NewMemoryAdapter(room)
	}

	r := server.NewAdapterRoomManager(newAdapter)
	rooms := server.NewChannelRoomManager(r)

	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()
	defer rooms.Close()

	events := make(chan server.RoomEvent)
	go func() {
		defer wg.Done()

		for {
			roomEvent, ok := <-rooms.RoomEventsChannel()
			if !ok {
				return
			}
			events <- roomEvent
		}
	}()

	adapter1, isNew := rooms.Enter("test")
	assert.True(t, isNew)
	_, ok := adapter1.(*server.MemoryAdapter)
	require.True(t, ok)
	assert.Equal(t, server.RoomEvent{"test", server.RoomEventTypeAdd}, <-events)

	adapter2, isNew := rooms.Enter("test")
	assert.False(t, isNew)
	assert.True(t, adapter1 == adapter2, "adapters should be the same")

	isRemoved := rooms.Exit("test")
	assert.False(t, isRemoved)

	adapter3, isNew := rooms.Enter("test")
	assert.False(t, isNew)
	assert.True(t, adapter1 == adapter3, "adapters should be the same")

	isRemoved = rooms.Exit("test")
	assert.False(t, isRemoved)

	isRemoved = rooms.Exit("test")
	assert.True(t, isRemoved)
	assert.Equal(t, server.RoomEvent{"test", server.RoomEventTypeRemove}, <-events)

	adapter4, isNew := rooms.Enter("test")
	assert.True(t, isNew)
	assert.True(t, adapter1 != adapter4, "adapters should NOT be the same")
	assert.Equal(t, server.RoomEvent{"test", server.RoomEventTypeAdd}, <-events)
}
