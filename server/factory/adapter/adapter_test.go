package adapter_test

import (
	"testing"

	"github.com/peer-calls/peer-calls/server/config"
	"github.com/peer-calls/peer-calls/server/factory/adapter"
	"github.com/peer-calls/peer-calls/server/ws/wsmemory"
	"github.com/peer-calls/peer-calls/server/ws/wsredis"
	"github.com/stretchr/testify/assert"
)

func TestNewAdapterFactory_redis(t *testing.T) {
	f := adapter.NewAdapterFactory(config.StoreConfig{
		Type: "redis",
		Redis: config.RedisConfig{
			Prefix: "peercalls",
			Host:   "localhost",
			Port:   6379,
		},
	})

	redisAdapter, ok := f.NewAdapter("test-room").(*wsredis.RedisAdapter)
	assert.True(t, ok)

	err := redisAdapter.Close()
	assert.Nil(t, err)
}

func TestNewAdapterFactory_memory(t *testing.T) {
	f := adapter.NewAdapterFactory(config.StoreConfig{
		Type: "memory",
	})

	_, ok := f.NewAdapter("test-room").(*wsmemory.MemoryAdapter)
	assert.True(t, ok)
}
