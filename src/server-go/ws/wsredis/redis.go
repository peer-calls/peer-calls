package wsredis

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/go-redis/redis/v7"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsmessage"
)

type Client interface {
	ID() string
	Messages() chan<- wsmessage.Message
}

type JSONMessage struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type Doner interface {
	Done()
}

var serializer wsmessage.ByteSerializer

type RedisAdapter struct {
	clientsMu *sync.RWMutex
	// contains local clients connected to current instance
	clients map[string]Client
	// contains IDs of all clients in room, including those from other instances
	allClients map[string]struct{}
	logger     *log.Logger
	prefix     string
	roomName   string
	pubRedis   *redis.Client // FIXME replace this with interface
	subRedis   *redis.Client
	channels   struct {
		roomChannel   string
		clientPattern string
	}
}

func getRoomChannelName(prefix string, roomName string) string {
	return prefix + ":room:" + roomName
}

func getClientChannelName(prefix string, roomName string, clientID string) string {
	return prefix + ":room:" + roomName + ":client:" + clientID
}

func NewRedisAdapter(
	pubRedis *redis.Client,
	subRedis *redis.Client,
	prefix string,
	roomName string,
) *RedisAdapter {
	var clientsMu sync.RWMutex

	adapter := RedisAdapter{
		clients:    map[string]Client{},
		allClients: map[string]struct{}{},
		clientsMu:  &clientsMu,
		logger:     log.New(os.Stdout, "wsredis ", log.LstdFlags),
		prefix:     prefix,
		roomName:   roomName,
		pubRedis:   pubRedis,
		subRedis:   subRedis,
	}

	adapter.channels.roomChannel = getRoomChannelName(prefix, roomName)
	adapter.channels.clientPattern = getClientChannelName(prefix, roomName, "*")

	return &adapter
}

func (a *RedisAdapter) Add(client Client) {
	clientID := client.ID()
	a.logger.Printf("Add clientID: %s to room: %s", clientID, a.roomName)
	a.clientsMu.Lock()
	a.Broadcast(wsmessage.NewMessageRoomJoin(clientID))
	a.clients[clientID] = client
	a.allClients[clientID] = struct{}{}
	a.clientsMu.Unlock()
}

func (a *RedisAdapter) Remove(clientID string) {
	a.logger.Printf("Remove clientID: %s from room: %s", clientID, a.roomName)
	a.clientsMu.Lock()
	if _, ok := a.clients[clientID]; ok {
		// can only remove clients connected to this adapter
		a.Broadcast(wsmessage.NewMessageRoomLeave(clientID))
		delete(a.clients, clientID)
		delete(a.allClients, clientID)
	}
	a.clientsMu.Unlock()
}

func (a *RedisAdapter) Clients() (clientIDs []string) {
	a.logger.Printf("Clients")
	a.clientsMu.RLock()
	for clientID := range a.allClients {
		clientIDs = append(clientIDs, clientID)
	}
	a.clientsMu.RUnlock()
	a.logger.Printf("Clients size: %d", len(clientIDs))
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
	msg := serializer.Deserialize([]byte(message))
	a.logger.Printf("handleMessage pattern: %s, channel: %s, type: %d, payload: %s", pattern, channel, msg.Type(), msg.Payload())
	switch {
	case channel == a.channels.roomChannel:
		// localBroadcast to all clients
		switch msg.Type() {
		case wsmessage.MessageTypeRoomJoin:
			a.clientsMu.Lock()
			clientID := string(msg.Payload())
			a.allClients[clientID] = struct{}{}
			a.localBroadcast(msg)
			a.clientsMu.Unlock()
		case wsmessage.MessageTypeRoomLeave:
			a.clientsMu.Lock()
			a.localBroadcast(msg)
			clientID := string(msg.Payload())
			delete(a.allClients, clientID)
			a.clientsMu.Unlock()
		default:
			a.clientsMu.RLock()
			a.localBroadcast(msg)
			a.clientsMu.RUnlock()
		}
	case pattern == a.channels.clientPattern:
		params := strings.Split(channel, ":")
		clientID := params[len(params)-1]
		a.clientsMu.RLock()
		a.localEmit(clientID, msg)
		a.clientsMu.RUnlock()
	}
	a.logger.Println("handleMessage done")
}

// Reads from subscribed channels and dispatches relevant messages to
// client websockets. This method blocks until the context is closed.
func (a *RedisAdapter) subscribe(ctx context.Context, ready func()) error {
	a.logger.Println("Subscribe", a.channels.roomChannel, a.channels.clientPattern)
	pubsub := a.subRedis.PSubscribe(a.channels.roomChannel, a.channels.clientPattern)
	ch := pubsub.Channel()

	pubsub.Ping()

	ready()
	a.logger.Println("Subscribe ready")

	for {
		select {
		case msg := <-ch:
			a.handleMessage(msg.Pattern, msg.Channel, msg.Payload)
		case <-ctx.Done():
			err := ctx.Err()
			a.logger.Println("Subscribe done", err)
			pubsub.Close()
			return err
		}
	}
}

func (a *RedisAdapter) Subscribe() (stop func() error) {
	var wg sync.WaitGroup
	wg.Add(1)
	errChan := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := a.subscribe(ctx, wg.Done)
		errChan <- err
		close(errChan)
	}()
	wg.Wait()
	return func() error {
		cancel()
		return <-errChan
	}
}

func (a *RedisAdapter) Broadcast(msg wsmessage.Message) {
	channel := a.channels.roomChannel
	a.logger.Printf("Broadcast type: %d, payload: %s to %s", msg.Type(), msg.Payload(), channel)
	a.pubRedis.Publish(channel, string(serializer.Serialize(msg)))
}

func (a *RedisAdapter) localBroadcast(msg wsmessage.Message) {
	a.logger.Printf("localBroadcast in room %s of message type: %d", a.roomName, msg.Type())
	for clientID := range a.clients {
		a.localEmit(clientID, msg)
	}
}

func (a *RedisAdapter) Emit(clientID string, msg wsmessage.Message) {
	channel := getClientChannelName(a.prefix, a.roomName, clientID)
	a.logger.Printf("Emit clientID: %s, type: %d, payload: %s to %s", clientID, msg.Type(), msg, channel)
	a.pubRedis.Publish(channel, string(serializer.Serialize(msg)))
}

func (a *RedisAdapter) localEmit(clientID string, msg wsmessage.Message) {
	client, ok := a.clients[clientID]
	if !ok {
		a.logger.Printf("localEmit in room: %s  - no local clientID: %s", a.roomName, clientID)
		return
	}
	select {
	case client.Messages() <- msg:
		a.logger.Printf("localEmit in room: %s - sent to local clientID: %s", a.roomName, clientID)
	default:
		a.logger.Printf("localEmit in room: %s - buffer full for clientID: %s", a.roomName, clientID)
	}
}
