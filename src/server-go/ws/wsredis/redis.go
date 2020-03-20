package wsredis

import (
	"sync"

	"github.com/jeremija/peer-calls/src/server-go/ws"
)

type RedisAdapter struct {
	clientsMu *sync.RWMutex
	clients   map[string]ws.Client
}

func NewRedisAdapter() *RedisAdapter {
	var clientsMu sync.RWMutex
	return &RedisAdapter{
		clientsMu: &clientsMu,
		clients:   map[string]ws.Client{},
	}
}
