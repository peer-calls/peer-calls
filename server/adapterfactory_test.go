package server_test

import (
	"testing"

	"github.com/peer-calls/peer-calls/server"
	"github.com/stretchr/testify/assert"
)

func TestNewAdapterFactory_redis(t *testing.T) {
	f := server.NewAdapterFactory(loggerFactory, server.StoreConfig{
		Type: "redis",
		Redis: server.RedisConfig{
			Prefix: "peercalls",
			Host:   "localhost",
			Port:   6379,
		},
	})

	redisAdapter, ok := f.NewAdapter("test-room").(*server.RedisAdapter)
	assert.True(t, ok)

	err := redisAdapter.Close()
	assert.Nil(t, err)
}

func TestNewAdapterFactory_memory(t *testing.T) {
	f := server.NewAdapterFactory(loggerFactory, server.StoreConfig{
		Type: "memory",
	})

	_, ok := f.NewAdapter("test-room").(*server.MemoryAdapter)
	assert.True(t, ok)
}
