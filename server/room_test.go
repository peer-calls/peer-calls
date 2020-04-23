package server_test

import (
	"testing"

	"github.com/peer-calls/peer-calls/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoomManager(t *testing.T) {
	var newAdapter server.NewAdapterFunc = func(room string) server.Adapter {
		return server.NewMemoryAdapter(room)
	}

	rooms := server.NewAdapterRoomManager(newAdapter)

	adapter1, ok := rooms.Enter("test").(*server.MemoryAdapter)
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
