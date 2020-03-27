package adapter_test

import (
	"testing"

	"github.com/jeremija/peer-calls/src/server-go/config"
	"github.com/jeremija/peer-calls/src/server-go/factory/adapter"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsmemory"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsredis"
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
