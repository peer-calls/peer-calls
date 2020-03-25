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
	logger   *log.Logger
	prefix   string
	room     string
	pubRedis *redis.Client // FIXME replace this with interface
	subRedis *redis.Client
	keys     struct {
		roomChannel   string
		roomClients   string
		clientPattern string
	}
}

func getRoomChannelName(prefix string, room string) string {
	return prefix + ":room:" + room + ":broadcast"
}

func getClientChannelName(prefix string, room string, clientID string) string {
	return prefix + ":room:" + room + ":client:" + clientID
}

func getRoomClientsName(prefix string, room string) string {
	return prefix + ":room:" + room + ":clients"
}

func NewRedisAdapter(
	pubRedis *redis.Client,
	subRedis *redis.Client,
	prefix string,
	room string,
) *RedisAdapter {
	var clientsMu sync.RWMutex

	adapter := RedisAdapter{
		clients:   map[string]Client{},
		clientsMu: &clientsMu,
		logger:    log.New(os.Stdout, "wsredis ", log.LstdFlags),
		prefix:    prefix,
		room:      room,
		pubRedis:  pubRedis,
		subRedis:  subRedis,
	}

	adapter.keys.roomChannel = getRoomChannelName(prefix, room)
	adapter.keys.clientPattern = getClientChannelName(prefix, room, "*")
	adapter.keys.roomClients = getRoomClientsName(prefix, room)

	return &adapter
}

func (a *RedisAdapter) Add(client Client) {
	clientID := client.ID()
	a.logger.Printf("Add clientID: %s to room: %s", clientID, a.room)
	a.clientsMu.Lock()
	a.Broadcast(wsmessage.NewMessageRoomJoin(a.room, clientID))
	a.clients[clientID] = client
	a.logger.Printf("Add clientID: %s to room: %s done", clientID, a.room)
	a.clientsMu.Unlock()
}

func (a *RedisAdapter) Remove(clientID string) {
	a.logger.Printf("Remove clientID: %s from room: %s", clientID, a.room)
	a.clientsMu.Lock()
	if _, ok := a.clients[clientID]; ok {
		// can only remove clients connected to this adapter
		if err := a.pubRedis.HDel(a.keys.roomClients, clientID).Err(); err != nil {
			a.logger.Printf("Error deleting clientID from all clients: %s", err)
		}
		a.Broadcast(wsmessage.NewMessageRoomLeave(a.room, clientID))
		delete(a.clients, clientID)
	}
	a.logger.Printf("Remove clientID: %s from room: %s done", clientID, a.room)
	a.clientsMu.Unlock()
}

// Returns IDs of all known clients connected to this room
func (a *RedisAdapter) Clients() (clientIDs []string) {
	a.logger.Printf("Clients")

	r := a.pubRedis.HGetAll(a.keys.roomClients)
	allClients, err := r.Result()

	if err != nil {
		a.logger.Printf("Error retrieving clients in room: %s, reason: %s", a.room, err)
	}

	clientIDs = make([]string, len(allClients))
	i := 0
	for clientID := range allClients {
		clientIDs[i] = clientID
		i++
	}
	a.logger.Printf("Clients size: %d", len(clientIDs))
	return
}

// Returns count of all known clients connected to this room
func (a *RedisAdapter) Size() (size int) {
	return len(a.Clients())
}

func (a *RedisAdapter) handleMessage(
	pattern string,
	channel string,
	message string,
) {
	msg := serializer.Deserialize([]byte(message))
	a.logger.Printf("handleMessage pattern: %s, channel: %s, type: %s, payload: %s", pattern, channel, msg.Type(), msg.Payload())
	switch {
	case channel == a.keys.roomChannel:
		// localBroadcast to all clients
		switch msg.Type() {
		case wsmessage.MessageTypeRoomJoin:
			a.clientsMu.Lock()
			clientID := string(msg.Payload())
			a.pubRedis.HSet(a.keys.roomClients, clientID, "")
			a.localBroadcast(msg)
			a.clientsMu.Unlock()
		case wsmessage.MessageTypeRoomLeave:
			a.clientsMu.Lock()
			a.localBroadcast(msg)
			clientID := string(msg.Payload())
			a.pubRedis.HDel(a.keys.roomClients, clientID)
			a.clientsMu.Unlock()
		default:
			a.clientsMu.RLock()
			a.localBroadcast(msg)
			a.clientsMu.RUnlock()
		}
	case pattern == a.keys.clientPattern:
		params := strings.Split(channel, ":")
		clientID := params[len(params)-1]
		a.clientsMu.RLock()
		a.localEmit(clientID, msg)
		a.clientsMu.RUnlock()
	}
	a.logger.Println("handleMessage done")
}

// Reads from subscribed keys and dispatches relevant messages to
// client websockets. This method blocks until the context is closed.
func (a *RedisAdapter) subscribe(ctx context.Context, ready func()) error {
	a.logger.Println("Subscribe", a.keys.roomChannel, a.keys.clientPattern)
	pubsub := a.subRedis.PSubscribe(a.keys.roomChannel, a.keys.clientPattern)
	ch := pubsub.ChannelWithSubscriptions(100)

	isReady := false

	for {
		select {
		case msg := <-ch:
			switch msg := msg.(type) {
			case *redis.Subscription:
				if !isReady {
					isReady = true
					ready()
				}
			case *redis.Message:
				a.handleMessage(msg.Pattern, msg.Channel, msg.Payload)
			}
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
	channel := a.keys.roomChannel
	a.logger.Printf("Broadcast type: %s, payload: %s to %s", msg.Type(), msg.Payload(), channel)
	a.pubRedis.Publish(channel, string(serializer.Serialize(msg)))
}

func (a *RedisAdapter) localBroadcast(msg wsmessage.Message) {
	a.logger.Printf("localBroadcast in room %s of message type: %s", a.room, msg.Type())
	for clientID := range a.clients {
		a.localEmit(clientID, msg)
	}
}

func (a *RedisAdapter) Emit(clientID string, msg wsmessage.Message) {
	channel := getClientChannelName(a.prefix, a.room, clientID)
	a.logger.Printf("Emit clientID: %s, type: %s, payload: %s to %s", clientID, msg.Type(), msg, channel)
	a.pubRedis.Publish(channel, string(serializer.Serialize(msg)))
}

func (a *RedisAdapter) localEmit(clientID string, msg wsmessage.Message) {
	client, ok := a.clients[clientID]
	if !ok {
		a.logger.Printf("localEmit in room: %s  - no local clientID: %s", a.room, clientID)
		return
	}
	select {
	case client.Messages() <- msg:
		a.logger.Printf("localEmit in room: %s - sent to local clientID: %s", a.room, clientID)
	default:
		a.logger.Printf("localEmit in room: %s - buffer full for clientID: %s", a.room, clientID)
	}
}
