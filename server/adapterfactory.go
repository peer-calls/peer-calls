package server

import (
	"net"
	"strconv"

	"github.com/go-redis/redis/v7"
	"github.com/juju/errors"
)

type AdapterFactory struct {
	pubClient *redis.Client
	subClient *redis.Client

	NewAdapter func(room string) Adapter
}

func NewAdapterFactory(
	loggerFactory LoggerFactory,
	c StoreConfig,
) *AdapterFactory {
	log := loggerFactory.GetLogger("adapterfactory")
	f := AdapterFactory{}

	switch c.Type {
	case StoreTypeRedis:
		addr := net.JoinHostPort(c.Redis.Host, strconv.Itoa(c.Redis.Port))
		prefix := c.Redis.Prefix
		log.Printf("Using RedisAdapter: %s with prefix %s", addr, prefix)

		f.pubClient = redis.NewClient(&redis.Options{
			Addr: addr,
		})

		f.subClient = redis.NewClient(&redis.Options{
			Addr: addr,
		})

		f.NewAdapter = func(room string) Adapter {
			return NewRedisAdapter(loggerFactory, f.pubClient, f.subClient, prefix, room)
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
