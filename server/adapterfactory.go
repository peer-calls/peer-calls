package server

import (
	"log"
	"net"
	"strconv"

	"github.com/go-redis/redis/v7"
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
)

type AdapterFactory struct {
	pubClient *redis.Client
	subClient *redis.Client

	NewAdapter func(room string) Adapter
}

func NewAdapterFactory(l logger.Logger, c StoreConfig) *AdapterFactory {
	l = l.WithNamespaceAppended("adapterfactory")
	f := AdapterFactory{}

	switch c.Type {
	case StoreTypeRedis:
		addr := net.JoinHostPort(c.Redis.Host, strconv.Itoa(c.Redis.Port))
		prefix := c.Redis.Prefix
		l.Info("Using RedisAdapter", logger.Ctx{
			"remote_addr": addr,
			"prefix":      prefix,
		})

		f.pubClient = redis.NewClient(&redis.Options{
			Addr: addr,
		})

		f.subClient = redis.NewClient(&redis.Options{
			Addr: addr,
		})

		f.NewAdapter = func(room string) Adapter {
			return NewRedisAdapter(l, f.pubClient, f.subClient, prefix, room)
		}
	default:
		log.Printf("Using MemoryAdapter")

		f.NewAdapter = func(room string) Adapter {
			return NewMemoryAdapter(room)
		}
	}

	return &f
}

func (a *AdapterFactory) Close() (err error) {
	var errs MultiErrorHandler

	if a.pubClient != nil {
		errs.Add(errors.Trace(a.pubClient.Close()))
	}

	if a.subClient != nil {
		errs.Add(errors.Trace(a.subClient.Close()))
	}

	return errors.Trace(errs.Err())
}
