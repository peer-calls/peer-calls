package wsredis

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/go-redis/redis/v7"
)

type Client interface {
	ID() string
	Messages() chan<- []byte
}

type JSONMessage struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type RedisAdapter struct {
	clientsMu *sync.RWMutex
	// contains local clients connected to current instance
	clients map[string]Client
	// contains IDs of all clients in room, including those from other instances
	allClients map[string]struct{}
	logger     *log.Logger
	prefix     string
	roomName   string
	redis      *redis.Client // FIXME replace this with interface
	patterns   struct {
		room      string
		roomJoin  string
		roomLeave string
		client    string
	}
}

func getRoomChannelName(prefix string, roomName string) string {
	return prefix + ":room:" + roomName + ":broadcast"
}

func getRoomJoinChannelName(prefix string, roomName string) string {
	return getRoomChannelName(prefix, roomName) + ":join"
}

func getRoomLeaveChannelName(prefix string, roomName string) string {
	return getRoomChannelName(prefix, roomName) + ":leave"
}

func getClientChannelName(prefix string, roomName string, clientID string) string {
	return prefix + ":room:" + roomName + ":client:" + clientID
}

func NewRedisAdapter(redis *redis.Client, prefix string, roomName string) *RedisAdapter {
	var clientsMu sync.RWMutex

	adapter := RedisAdapter{
		clients:   map[string]Client{},
		clientsMu: &clientsMu,
		logger:    log.New(os.Stdout, "WSREDIS ", log.LstdFlags),
		prefix:    prefix,
		redis:     redis,
	}

	adapter.patterns.room = getRoomChannelName(prefix, roomName)
	adapter.patterns.roomJoin = getRoomJoinChannelName(prefix, roomName)
	adapter.patterns.roomLeave = getRoomLeaveChannelName(prefix, roomName)
	adapter.patterns.client = getClientChannelName(prefix, roomName, "*")

	return &adapter
}

func (a *RedisAdapter) Add(client Client) {
	a.clientsMu.Lock()
	clientID := client.ID()
	a.logger.Printf("Room: %s Add: %s", a.roomName, clientID)
	a.redis.Publish(a.patterns.roomJoin, clientID)
	a.clients[clientID] = client
	a.clientsMu.Unlock()
}

func (a *RedisAdapter) Remove(clientID string) {
	a.logger.Printf("Room: %s Remove: %s", a.roomName, clientID)
	a.redis.Publish(a.patterns.roomLeave, clientID)
}

func (a *RedisAdapter) Clients() (clientIDs []string) {
	a.clientsMu.RLock()
	for clientID := range a.allClients {
		clientIDs = append(clientIDs, clientID)
	}
	a.clientsMu.RUnlock()
	return
}

func (a *RedisAdapter) Size() int {
	a.clientsMu.RLock()
	defer a.clientsMu.RUnlock()
	return len(a.clients)
}

func (a *RedisAdapter) handleMessage(
	pattern string,
	channel string,
	message string,
) {
	switch {
	case channel == a.patterns.room:
		// broadcast
		a.clientsMu.RLock()
		for _, client := range a.clients {
			client.Messages() <- []byte(message)
		}
		a.clientsMu.RUnlock()
	case channel == a.patterns.roomJoin:
		a.clientsMu.Lock()
		clientID := message
		a.allClients[clientID] = struct{}{}
		a.clientsMu.Unlock()
	case channel == a.patterns.roomLeave:
		a.clientsMu.Lock()
		clientID := message
		delete(a.clients, clientID)
		delete(a.allClients, clientID)
		a.clientsMu.Unlock()
	case pattern == a.patterns.client:
		params := strings.Split(channel, ":")
		clientID := params[len(params)-1]

		a.clientsMu.RLock()
		client, ok := a.clients[clientID]
		if ok {
			client.Messages() <- []byte(message)
		}
		a.clientsMu.RUnlock()
	}
}

// Reads from subscribed channels and dispatches relevant messages to
// client websockets. This method blocks until the context is closed.
func (a *RedisAdapter) Subscribe(ctx context.Context) error {
	pubsub := a.redis.PSubscribe(a.patterns.room, a.patterns.client)
	ch := pubsub.Channel()

	for {
		select {
		case msg := <-ch:
			a.handleMessage(msg.Pattern, msg.Channel, msg.Payload)
		case <-ctx.Done():
			pubsub.Close()
			return ctx.Err()
		}
	}
}

func (a *RedisAdapter) Broadcast(msg []byte) {
	channel := a.patterns.room
	a.redis.Publish(channel, string(msg))
	// TODO publish to local clients if any
}

func (a *RedisAdapter) Emit(clientID string, msg []byte) {
	channel := getClientChannelName(a.prefix, a.roomName, clientID)
	a.redis.Publish(channel, string(msg))
	// TODO publish to local clients if any
}
