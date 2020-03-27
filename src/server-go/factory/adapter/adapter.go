package adapter

import (
	"net"
	"strconv"

	"github.com/go-redis/redis/v7"
	"github.com/jeremija/peer-calls/src/server-go/config"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsadapter"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsmemory"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsredis"
)

type AdapterFactory struct {
	pubClient *redis.Client
	subClient *redis.Client

	NewAdapter func(room string) wsadapter.Adapter
}

func NewAdapterFactory(c config.StoreConfig) *AdapterFactory {
	f := AdapterFactory{}

	switch c.Type {
	case config.StoreTypeRedis:
		f.pubClient = redis.NewClient(&redis.Options{
			Addr: net.JoinHostPort(c.Redis.Host, strconv.Itoa(c.Redis.Port)),
		})
		f.subClient = redis.NewClient(&redis.Options{
			Addr: net.JoinHostPort(c.Redis.Host, strconv.Itoa(c.Redis.Port)),
		})
		prefix := c.Redis.Prefix
		f.NewAdapter = func(room string) wsadapter.Adapter {
			return wsredis.NewRedisAdapter(f.pubClient, f.subClient, prefix, room)
		}
	default:
		f.NewAdapter = func(room string) wsadapter.Adapter {
			return wsmemory.NewMemoryAdapter(room)
		}
	}

	return &f
}

func (a *AdapterFactory) Close() (err error) {
	if a.pubClient != nil {
		err = a.pubClient.Close()
	}
	if a.subClient != nil {
		if subError := a.subClient.Close(); subError != nil && err == nil {
			err = subError
		}
	}
	return
}
