package server

import (
	"net"
	"strconv"

	"github.com/go-redis/redis/v7"
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/logger"
)

type AdapterFactory struct {
	pubClient *redis.Client
	subClient *redis.Client

	NewAdapter func(room identifiers.RoomID) Adapter
}

func NewAdapterFactory(log logger.Logger, c StoreConfig) *AdapterFactory {
	log = log.WithNamespaceAppended("adapterfactory")
	f := AdapterFactory{}

	switch c.Type {
	case StoreTypeRedis:
		addr := net.JoinHostPort(c.Redis.Host, strconv.Itoa(c.Redis.Port))
		prefix := c.Redis.Prefix

		log.Info("Using RedisAdapter", logger.Ctx{
			"remote_addr": addr,
			"prefix":      prefix,
		})

		f.pubClient = redis.NewClient(&redis.Options{
			Addr: addr,
		})

		f.subClient = redis.NewClient(&redis.Options{
			Addr: addr,
		})

		f.NewAdapter = func(room identifiers.RoomID) Adapter {
			return NewRedisAdapter(log, f.pubClient, f.subClient, prefix, room)
		}
	default:
		log.Info("Using MemoryAdapter", nil)

		f.NewAdapter = func(room identifiers.RoomID) Adapter {
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
