package wsredis

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/go-redis/redis/v7"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsmessage"
)

type Client interface {
	ID() string
	WriteChannel() chan<- wsmessage.Message
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
	pubRedis *redis.Client // TODO replace this with interface
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

func (a *RedisAdapter) Add(client Client) (err error) {
	clientID := client.ID()
	a.logger.Printf("Add clientID: %s to room: %s", clientID, a.room)
	a.clientsMu.Lock()
	err = a.Broadcast(wsmessage.NewMessageRoomJoin(a.room, clientID))
	if err == nil {
		a.clients[clientID] = client
		a.logger.Printf("Add clientID: %s to room: %s done", clientID, a.room)
	}
	a.clientsMu.Unlock()
	return
}

func (a *RedisAdapter) Remove(clientID string) (err error) {
	a.logger.Printf("Remove clientID: %s from room: %s", clientID, a.room)
	a.clientsMu.Lock()
	if _, ok := a.clients[clientID]; ok {
		// can only remove clients connected to this adapter
		if err = a.pubRedis.HDel(a.keys.roomClients, clientID).Err(); err != nil {
			a.logger.Printf("Error deleting clientID from all clients: %s", err)
		} else {
			err = a.Broadcast(wsmessage.NewMessageRoomLeave(a.room, clientID))
		}
		delete(a.clients, clientID)
	}
	a.logger.Printf("Remove clientID: %s from room: %s done (err: %s)", clientID, a.room, err)
	a.clientsMu.Unlock()
	return
}

// Returns IDs of all known clients connected to this room
func (a *RedisAdapter) Clients() (clientIDs []string, err error) {
	a.logger.Printf("Clients")

	r := a.pubRedis.HGetAll(a.keys.roomClients)
	allClients, err := r.Result()

	if err != nil {
		err = fmt.Errorf("Error retrieving clients in room: %s, reason: %w", a.room, err)
		a.logger.Printf("Error retrieving clients in room: %s, reason: %s", a.room, err)
		return
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
func (a *RedisAdapter) Size() (size int, err error) {
	c, err := a.Clients()
	return len(c), err
}

func (a *RedisAdapter) handleMessage(
	pattern string,
	channel string,
	message string,
) error {
	msg, err := serializer.Deserialize([]byte(message))
	if err != nil {
		return fmt.Errorf("RedisAdapter.handleMessage error deserializing redis subscription: %w", err)
	}
	a.logger.Printf("RedisAdapter.handleMessage pattern: %s, channel: %s, type: %s, payload: %s", pattern, channel, msg.Type(), msg.Payload())
	switch {
	case channel == a.keys.roomChannel:
		// localBroadcast to all clients
		switch msg.Type() {
		case wsmessage.MessageTypeRoomJoin:
			a.clientsMu.Lock()
			clientID := string(msg.Payload())
			err = a.pubRedis.HSet(a.keys.roomClients, clientID, "").Err()
			if err == nil {
				err = a.localBroadcast(msg)
			}
			a.clientsMu.Unlock()
		case wsmessage.MessageTypeRoomLeave:
			a.clientsMu.Lock()
			err = a.localBroadcast(msg)
			if err == nil {
				clientID := string(msg.Payload())
				err = a.pubRedis.HDel(a.keys.roomClients, clientID).Err()
			}
			a.clientsMu.Unlock()
		default:
			a.clientsMu.RLock()
			err = a.localBroadcast(msg)
			a.clientsMu.RUnlock()
		}
	case pattern == a.keys.clientPattern:
		params := strings.Split(channel, ":")
		clientID := params[len(params)-1]
		a.clientsMu.RLock()
		err = a.localEmit(clientID, msg)
		a.clientsMu.RUnlock()
	}
	a.logger.Printf("RedisAdapter.handleMessage done (err: %s)", err)
	return err
}

// Reads from subscribed keys and dispatches relevant messages to
// client websockets. This method blocks until the context is closed.
func (a *RedisAdapter) subscribe(ctx context.Context, ready func()) error {
	a.logger.Println("Subscribe", a.keys.roomChannel, a.keys.clientPattern)
	pubsub := a.subRedis.PSubscribe(a.keys.roomChannel, a.keys.clientPattern)
	defer pubsub.Close()

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
				err := a.handleMessage(msg.Pattern, msg.Channel, msg.Payload)
				if err != nil {
					return fmt.Errorf("Error handling message: %w", err)
				}
			}
		case <-ctx.Done():
			err := ctx.Err()
			a.logger.Println("Subscribe done", err)
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

func (a *RedisAdapter) publish(channel string, msg wsmessage.Message) error {
	data, err := serializer.Serialize(msg)
	if err != nil {
		return fmt.Errorf("RedisAdapter.publish - error serializing message: %w", err)
	}
	return a.pubRedis.Publish(channel, string(data)).Err()
}

func (a *RedisAdapter) Broadcast(msg wsmessage.Message) error {
	channel := a.keys.roomChannel
	a.logger.Printf("Broadcast type: %s, payload: %s to %s", msg.Type(), msg.Payload(), channel)
	return a.publish(channel, msg)
}

func (a *RedisAdapter) localBroadcast(msg wsmessage.Message) (err error) {
	a.logger.Printf("RedisAdapter.localBroadcast in room %s of message type: %s", a.room, msg.Type())
	for clientID := range a.clients {
		if emitErr := a.localEmit(clientID, msg); emitErr != nil && err == nil {
			err = emitErr
		}
	}
	return
}

func (a *RedisAdapter) Emit(clientID string, msg wsmessage.Message) error {
	channel := getClientChannelName(a.prefix, a.room, clientID)
	a.logger.Printf("Emit clientID: %s, type: %s, payload: %s to %s", clientID, msg.Type(), msg, channel)
	data, err := serializer.Serialize(msg)
	if err != nil {
		return fmt.Errorf("RedisAdapter.Emit - error serializing message: %w", err)
	}
	return a.pubRedis.Publish(channel, string(data)).Err()
}

func (a *RedisAdapter) localEmit(clientID string, msg wsmessage.Message) error {
	client, ok := a.clients[clientID]
	if !ok {
		return fmt.Errorf("RedisAdapter.localEmit in room: %s - no local clientID: %s", a.room, clientID)
	}
	select {
	case client.WriteChannel() <- msg:
		return nil
	default:
		return fmt.Errorf("RedisAdapter.localEmit in room: %s - buffer full for clientID: %s", a.room, clientID)
	}
}
