package adapter

import (
	"net"
	"strconv"

	"github.com/go-redis/redis/v7"
	"github.com/peer-calls/peer-calls/server/config"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/ws/wsadapter"
	"github.com/peer-calls/peer-calls/server/ws/wsmemory"
	"github.com/peer-calls/peer-calls/server/ws/wsredis"
)

type AdapterFactory struct {
	pubClient *redis.Client
	subClient *redis.Client

	NewAdapter func(room string) wsadapter.Adapter
}

var log = logger.GetLogger("adapterfactory")

func NewAdapterFactory(c config.StoreConfig) *AdapterFactory {
	f := AdapterFactory{}

	switch c.Type {
	case config.StoreTypeRedis:
		addr := net.JoinHostPort(c.Redis.Host, strconv.Itoa(c.Redis.Port))
		prefix := c.Redis.Prefix
		log.Printf("Using RedisAdapter: %s with prefix %s", addr, prefix)
		f.pubClient = redis.NewClient(&redis.Options{
			Addr: addr,
		})
		f.subClient = redis.NewClient(&redis.Options{
			Addr: addr,
		})
		f.NewAdapter = func(room string) wsadapter.Adapter {
			return wsredis.NewRedisAdapter(f.pubClient, f.subClient, prefix, room)
		}
	default:
		log.Printf("Using MemoryAdapter")
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
