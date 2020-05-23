package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/go-redis/redis/v7"
)

type JSONMessage struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type Doner interface {
	Done()
}

type RedisAdapter struct {
	log          Logger
	serializer   Serializer
	deserializer Deserializer

	clientsMu *sync.RWMutex
	// contains local clients connected to current instance
	clients map[string]ClientWriter
	// contains IDs of all clients in room, including those from other instances
	prefix   string
	room     string
	pubRedis *redis.Client // TODO replace this with interface
	subRedis *redis.Client
	keys     struct {
		roomChannel   string
		roomClients   string
		clientPattern string
	}
	stop func() error
}

func getRoomChannelName(prefix string, room string) string {
	// TODO escape room name, what if it has ":" in the name?
	return prefix + ":room:" + room + ":broadcast"
}

func getClientChannelName(prefix string, room string, clientID string) string {
	// TODO escape room name, what if it has ":" in the name?
	return prefix + ":room:" + room + ":client:" + clientID
}

func getRoomClientsName(prefix string, room string) string {
	// TODO escape room name, what if it has ":" in the name?
	return prefix + ":room:" + room + ":clients"
}

func NewRedisAdapter(
	loggerFactory LoggerFactory,
	pubRedis *redis.Client,
	subRedis *redis.Client,
	prefix string,
	room string,
) *RedisAdapter {
	var clientsMu sync.RWMutex
	var byteSerializer ByteSerializer

	adapter := RedisAdapter{
		log:          loggerFactory.GetLogger("redis"),
		serializer:   byteSerializer,
		deserializer: byteSerializer,
		clients:      map[string]ClientWriter{},
		clientsMu:    &clientsMu,
		prefix:       prefix,
		room:         room,
		pubRedis:     pubRedis,
		subRedis:     subRedis,
		stop:         nil,
	}

	adapter.keys.roomChannel = getRoomChannelName(prefix, room)
	adapter.keys.clientPattern = getClientChannelName(prefix, room, "*")
	adapter.keys.roomClients = getRoomClientsName(prefix, room)

	adapter.subscribeUntilReady()

	return &adapter
}

func (a *RedisAdapter) Add(client ClientWriter) (err error) {
	clientID := client.ID()
	a.log.Printf("Add clientID: %s to room: %s", clientID, a.room)
	a.clientsMu.Lock()
	err = a.Broadcast(NewMessageRoomJoin(a.room, clientID, client.Metadata()))
	if err == nil {
		a.clients[clientID] = client
		a.log.Printf("Add clientID: %s to room: %s done", clientID, a.room)
	}
	a.clientsMu.Unlock()
	return
}

func (a *RedisAdapter) Remove(clientID string) (err error) {
	a.clientsMu.Lock()
	if _, ok := a.clients[clientID]; ok {
		err = a.remove(clientID)
	}
	a.clientsMu.Unlock()
	return
}

func (a *RedisAdapter) removeAll() (err error) {
	for clientID := range a.clients {
		if removeErr := a.remove(clientID); removeErr != nil && err == nil {
			err = removeErr
		}
	}
	return
}

func (a *RedisAdapter) remove(clientID string) (err error) {
	a.log.Printf("Remove clientID: %s from room: %s", clientID, a.room)
	// can only remove clients connected to this adapter
	if err = a.pubRedis.HDel(a.keys.roomClients, clientID).Err(); err != nil {
		a.log.Printf("Error deleting clientID from all clients: %s", err)
	}
	delete(a.clients, clientID)
	err = a.Broadcast(NewMessageRoomLeave(a.room, clientID))
	a.log.Printf("Remove clientID: %s from room: %s done (err: %s)", clientID, a.room, err)
	return
}

func (a *RedisAdapter) Metadata(clientID string) (metadata string, ok bool) {
	metadata, err := a.pubRedis.HGet(a.keys.roomClients, clientID).Result()
	return metadata, err == nil
}

func (a *RedisAdapter) SetMetadata(clientID string, metadata string) (ok bool) {
	_, err := a.pubRedis.HSet(a.keys.roomClients, clientID, metadata).Result()
	a.log.Printf("SetMetadata for clientID: %s, metadata: %s (err: %s)", clientID, metadata, err)
	return err == nil
}

// Returns IDs of all known clients connected to this room
func (a *RedisAdapter) Clients() (map[string]string, error) {
	a.log.Printf("Clients")

	r := a.pubRedis.HGetAll(a.keys.roomClients)
	allClients, err := r.Result()

	if err != nil {
		err = fmt.Errorf("Error retrieving clients in room: %s, reason: %w", a.room, err)
		a.log.Printf("Error retrieving clients in room: %s, reason: %s", a.room, err)
		return allClients, err
	}

	a.log.Printf("Clients size: %d", len(allClients))
	return allClients, nil
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
	msg, err := a.deserializer.Deserialize([]byte(message))
	if err != nil {
		return fmt.Errorf("RedisAdapter.handleMessage error deserializing redis subscription: %w", err)
	}
	a.log.Printf("RedisAdapter.handleMessage pattern: %s, channel: %s, type: %s", pattern, channel, msg.Type)
	switch {
	case channel == a.keys.roomChannel:
		// localBroadcast to all clients
		switch msg.Type {
		case MessageTypeRoomJoin:
			a.clientsMu.Lock()
			payload, ok := msg.Payload.(map[string]interface{})
			if ok {
				err = a.pubRedis.HSet(a.keys.roomClients, payload["clientID"], payload["metadata"]).Err()
				if err == nil {
					err = a.localBroadcast(msg)
				}
			}
			a.clientsMu.Unlock()
		case MessageTypeRoomLeave:
			a.clientsMu.Lock()
			err = a.localBroadcast(msg)
			if err == nil {
				clientID, ok := msg.Payload.(string)
				if ok {
					err = a.pubRedis.HDel(a.keys.roomClients, clientID).Err()
				}
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
	a.log.Printf("RedisAdapter.handleMessage done (err: %s)", err)
	return err
}

// Reads from subscribed keys and dispatches relevant messages to
// client websockets. This method blocks until the context is closed.
func (a *RedisAdapter) subscribe(ctx context.Context, ready func()) error {
	a.log.Println("Subscribe", a.keys.roomChannel, a.keys.clientPattern)
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
					a.log.Printf("Error handling message: %s", err)
				}
			}
		case <-ctx.Done():
			err := ctx.Err()
			a.log.Println("Subscribe done", err)
			return err
		}
	}
}

func (a *RedisAdapter) subscribeUntilReady() {
	var wg sync.WaitGroup
	wg.Add(1)
	subscribeEndedChan := make(chan struct{})
	var subscribeEndErr error
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		subscribeEndErr = a.subscribe(ctx, wg.Done)
		close(subscribeEndedChan)
	}()
	wg.Wait()

	a.stop = func() error {
		cancel()
		<-subscribeEndedChan
		return subscribeEndErr
	}
}

// Close closes the subscription, but not the redis clients
func (a *RedisAdapter) Close() error {
	var errs []error
	if a.stop != nil {
		if err := a.stop(); !errors.Is(err, context.Canceled) {
			errs = append(errs, err)
		}
	}
	a.clientsMu.Lock()
	defer a.clientsMu.Unlock()
	if err := a.removeAll(); err != nil {
		errs = append(errs, err)
	}
	return firstError(errs...)
}

func (a *RedisAdapter) publish(channel string, msg Message) error {
	data, err := a.serializer.Serialize(msg)
	if err != nil {
		return fmt.Errorf("RedisAdapter.publish - error serializing message: %w", err)
	}
	return a.pubRedis.Publish(channel, string(data)).Err()
}

func (a *RedisAdapter) Broadcast(msg Message) error {
	channel := a.keys.roomChannel
	a.log.Printf("RedisAdapter.Broadcast type: %s to %s", msg.Type, channel)
	return a.publish(channel, msg)
}

func (a *RedisAdapter) localBroadcast(msg Message) (err error) {
	a.log.Printf("RedisAdapter.localBroadcast in room %s of message type: %s", a.room, msg.Type)
	for clientID := range a.clients {
		if emitErr := a.localEmit(clientID, msg); emitErr != nil && err == nil {
			err = emitErr
		}
	}
	return
}

func (a *RedisAdapter) Emit(clientID string, msg Message) error {
	channel := getClientChannelName(a.prefix, a.room, clientID)
	a.log.Printf("Emit clientID: %s, type: %s to %s", clientID, msg.Type, channel)
	data, err := a.serializer.Serialize(msg)
	if err != nil {
		return fmt.Errorf("RedisAdapter.Emit - error serializing message: %w", err)
	}
	return a.pubRedis.Publish(channel, string(data)).Err()
}

func (a *RedisAdapter) localEmit(clientID string, msg Message) error {
	a.log.Printf("RedisAdapter.localEmit clientID: %s, type: %s", clientID, msg.Type)
	client, ok := a.clients[clientID]
	if !ok {
		return fmt.Errorf("RedisAdapter.localEmit in room: %s - no local clientID: %s", a.room, clientID)
	}
	err := client.Write(msg)
	if err != nil {
		return fmt.Errorf("RedisAdapter.localEmit in room: %s - error %w for clientID: %s", a.room, err, clientID)
	}
	return nil
}
