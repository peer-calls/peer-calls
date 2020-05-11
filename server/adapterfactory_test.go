package server_test

import (
	"testing"

	"github.com/peer-calls/peer-calls/server"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestNewAdapterFactory_redis(t *testing.T) {
	defer goleak.VerifyNone(t)
	f := server.NewAdapterFactory(loggerFactory, server.StoreConfig{
		Type: "redis",
		Redis: server.RedisConfig{
			Prefix: "peercalls",
			Host:   "localhost",
			Port:   6379,
		},
	})
	defer f.Close()

	redisAdapter, ok := f.NewAdapter("test-room").(*server.RedisAdapter)
	assert.True(t, ok)

	err := redisAdapter.Close()
	assert.Nil(t, err)
}

func TestNewAdapterFactory_memory(t *testing.T) {
	defer goleak.VerifyNone(t)
	f := server.NewAdapterFactory(loggerFactory, server.StoreConfig{
		Type: "memory",
	})
	defer f.Close()

	_, ok := f.NewAdapter("test-room").(*server.MemoryAdapter)
	assert.True(t, ok)
}
