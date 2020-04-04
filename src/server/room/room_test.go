package room_test

import (
	"testing"

	"github.com/jeremija/peer-calls/src/server/room"
	"github.com/jeremija/peer-calls/src/server/ws/wsadapter"
	"github.com/jeremija/peer-calls/src/server/ws/wsmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var newAdapter room.AdapterFactory = func(room string) wsadapter.Adapter {
	return wsmemory.NewMemoryAdapter(room)
}

func TestRoomManager(t *testing.T) {
	rooms := room.NewRoomManager(newAdapter)

	adapter1, ok := rooms.Enter("test").(*wsmemory.MemoryAdapter)
	require.True(t, ok)

	adapter2 := rooms.Enter("test")
	assert.True(t, adapter1 == adapter2, "adapters should be the same")

	rooms.Exit("test")
	adapter3 := rooms.Enter("test")
	assert.True(t, adapter1 == adapter3, "adapters should be the same")

	rooms.Exit("test")
	rooms.Exit("test")

	adapter4 := rooms.Enter("test")
	assert.True(t, adapter1 != adapter4, "adapters should NOT be the same")
}
