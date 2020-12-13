package server

import (
	"context"
	e "errors"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/juju/errors"
)

const (
	defaultSubscriptionTimeout     = 10 * time.Second
	defaultSubscriptionChannelSize = 100
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
	var (
		clientsMu      sync.RWMutex
		byteSerializer ByteSerializer
	)

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

	adapter.subscribeUntilReady(defaultSubscriptionTimeout)

	return &adapter
}

func (a *RedisAdapter) Add(client ClientWriter) (err error) {
	clientID := client.ID()
	a.log.Printf("Add clientID: %s to room: %s", clientID, a.room)

	a.clientsMu.Lock()
	a.clients[clientID] = client
	a.clientsMu.Unlock()

	err = a.Broadcast(NewMessageRoomJoin(a.room, clientID, client.Metadata()))
	if err != nil {
		return errors.Annotatef(err, "add client: %s", clientID)
	}

	a.log.Printf("Add clientID: %s to room: %s done", clientID, a.room)
	return nil
}

func (a *RedisAdapter) Remove(clientID string) error {
	a.clientsMu.Lock()
	_, ok := a.clients[clientID]
	delete(a.clients, clientID)
	a.clientsMu.Unlock()

	if !ok {
		return nil
	}

	err := a.remove(clientID)
	return errors.Annotatef(err, "remove client: %s", clientID)
}

func (a *RedisAdapter) removeAll(clientIDs []string) (err error) {
	var errs MultiErrorHandler

	for _, clientID := range clientIDs {
		if err := a.remove(clientID); err != nil {
			errs.Add(errors.Trace(err))
		}
	}

	return errors.Trace(errs.Err())
}

func (a *RedisAdapter) remove(clientID string) (err error) {
	var errs MultiErrorHandler

	a.log.Printf("Remove clientID: %s from room: %s", clientID, a.room)
	// can only remove clients connected to this adapter
	if err = a.pubRedis.HDel(a.keys.roomClients, clientID).Err(); err != nil {
		errs.Add(errors.Annotatef(err, "hdel %s %s", a.keys.roomClients, clientID))
	}

	if err = a.Broadcast(NewMessageRoomLeave(a.room, clientID)); err != nil {
		errs.Add(errors.Annotatef(err, "broadcast room leave %s %s", a.keys.roomClients, clientID))
	}

	return errors.Trace(errs.Err())
}

func (a *RedisAdapter) Metadata(clientID string) (metadata string, ok bool) {
	metadata, err := a.pubRedis.HGet(a.keys.roomClients, clientID).Result()

	return metadata, err == nil
}

func (a *RedisAdapter) SetMetadata(clientID string, metadata string) (ok bool) {
	_, err := a.pubRedis.HSet(a.keys.roomClients, clientID, metadata).Result()
	if err != nil {
		// FIXME return error
		a.log.Printf("Error SetMetadata for clientID: %s, metadata: %s", clientID, metadata, err)
	} else {
		a.log.Printf("SetMetadata for clientID: %s, metadata: %s", clientID, metadata)
	}

	return err == nil
}

// Returns IDs of all known clients connected to this room
func (a *RedisAdapter) Clients() (map[string]string, error) {
	a.log.Printf("Clients")

	r := a.pubRedis.HGetAll(a.keys.roomClients)

	allClients, err := r.Result()
	if err != nil {
		a.log.Printf("Error retrieving clients in room: %s, reason: %s", a.room, err)

		return allClients, errors.Annotatef(err, "clients in room: %s", a.room)
	}

	a.log.Printf("Clients size: %d", len(allClients))
	return allClients, nil
}

// Returns count of all known clients connected to this room
func (a *RedisAdapter) Size() (size int, err error) {
	c, err := a.Clients()

	return len(c), errors.Annotate(err, "size")
}

func (a *RedisAdapter) handleMessage(
	pattern string,
	channel string,
	message string,
) error {
	msg, err := a.deserializer.Deserialize([]byte(message))
	if err != nil {
		return errors.Annotate(err, "deserialize redis subscription")
	}

	a.log.Printf("RedisAdapter.handleMessage pattern: %s, channel: %s, type: %s", pattern, channel, msg.Type)

	handleRoomJoin := func() error {
		a.clientsMu.RLock()
		clients := a.localClients()
		a.clientsMu.RUnlock()

		payload, ok := msg.Payload.(map[string]interface{})
		if !ok {
			return errors.Errorf("room join: expected a map[string]interface{}, but got payload of type %T", msg.Payload)
		}

		err = a.pubRedis.HSet(a.keys.roomClients, payload["clientID"], payload["metadata"]).Err()
		if err != nil {
			return errors.Annotate(err, "room join")
		}

		err = a.localBroadcast(clients, msg)
		return errors.Annotate(err, "room join")
	}

	handleRoomLeave := func() error {
		a.clientsMu.RLock()
		clients := a.localClients()
		a.clientsMu.RUnlock()

		err = a.localBroadcast(clients, msg)
		if err != nil {
			return errors.Trace(err)
		}

		clientID, ok := msg.Payload.(string)
		if !ok {
			return errors.Errorf("room leave: expected a string, but got payload of type %T", msg.Payload)
		}

		err = a.pubRedis.HDel(a.keys.roomClients, clientID).Err()

		return errors.Annotate(err, "room leave")
	}

	handleRoomBroadcast := func() error {
		a.clientsMu.RLock()
		clients := a.localClients()
		a.clientsMu.RUnlock()

		err = a.localBroadcast(clients, msg)

		return errors.Annotate(err, "room broadcast")
	}

	handlePattern := func() error {
		params := strings.Split(channel, ":")
		clientID := params[len(params)-1]

		a.clientsMu.RLock()
		client, ok := a.clients[clientID]
		a.clientsMu.RUnlock()

		if !ok {
			return errors.Annotatef(err, "client %s not found", clientID)
		}

		err = a.localEmit(client, msg)
		return errors.Annotatef(err, "channel %s", channel)
	}

	switch {
	case channel == a.keys.roomChannel:
		// localBroadcast to all clients
		switch msg.Type {
		case MessageTypeRoomJoin:
			err = errors.Trace(handleRoomJoin())
		case MessageTypeRoomLeave:
			err = errors.Trace(handleRoomLeave())
		default:
			err = errors.Trace(handleRoomBroadcast())
		}
	case pattern == a.keys.clientPattern:
		err = errors.Trace(handlePattern())
	}

	a.log.Printf("RedisAdapter.handleMessage done (err: %s)", err)

	return errors.Trace(err)
}

// Reads from subscribed keys and dispatches relevant messages to
// client websockets. This method blocks until the context is closed.
func (a *RedisAdapter) subscribe(ctx context.Context, ready chan<- struct{}) error {
	a.log.Println("Subscribe", a.keys.roomChannel, a.keys.clientPattern)
	pubsub := a.subRedis.PSubscribe(a.keys.roomChannel, a.keys.clientPattern)

	defer pubsub.Close()

	ch := pubsub.ChannelWithSubscriptions(defaultSubscriptionChannelSize)

	isReady := false

	for {
		select {
		case msg := <-ch:
			switch msg := msg.(type) {
			case *redis.Subscription:
				if !isReady {
					isReady = true

					close(ready)
				}
			case *redis.Message:
				err := a.handleMessage(msg.Pattern, msg.Channel, msg.Payload)
				if err != nil {
					a.log.Printf("Error handling message: %+v", errors.Trace(err))
				}
			}
		case <-ctx.Done():
			err := ctx.Err()
			a.log.Println("Subscribe done", err)
			return errors.Trace(err)
		}
	}
}

func (a *RedisAdapter) subscribeUntilReady(timeout time.Duration) {
	var err error

	done := make(chan struct{})
	ready := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())

	a.stop = func() error {
		cancel()
		<-done

		return errors.Trace(err)
	}

	go func() {
		err = errors.Trace(a.subscribe(ctx, ready))

		close(done)
	}()

	var timeoutDoneCh <-chan struct{}

	if timeout > 0 {
		timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), timeout)
		defer timeoutCancel()

		timeoutDoneCh = timeoutCtx.Done()
	}

	select {
	case <-ready:
		// TODO perhaps it is not necessary to block here: an event with all users
		// could be sent to all connected clients immediately after the
		// subscription is completed.
		//
		// This would require some refactoring as currently the "users" event is
		// being sent only after "ready" event has been received.
	case <-timeoutDoneCh:
		cancel()
	}
}

// Close closes the subscription, but not the redis clients
func (a *RedisAdapter) Close() error {
	var errs MultiErrorHandler

	if a.stop != nil {
		if err := errors.Cause(a.stop()); !e.Is(err, context.Canceled) {
			errs.Add(errors.Trace(err))
		}
	}

	a.clientsMu.Lock()
	clientIDs := make([]string, 0, len(a.clients))

	for clientID := range a.clients {
		clientIDs = append(clientIDs, clientID)
		delete(a.clients, clientID)
	}
	a.clientsMu.Unlock()

	if err := a.removeAll(clientIDs); err != nil {
		errs.Add(errors.Trace(err))
	}

	return errors.Trace(errs.Err())
}

func (a *RedisAdapter) localClients() map[string]ClientWriter {
	clients := make(map[string]ClientWriter, len(a.clients))

	for k, v := range a.clients {
		clients[k] = v
	}

	return clients
}

func (a *RedisAdapter) publish(channel string, msg Message) error {
	data, err := a.serializer.Serialize(msg)
	if err != nil {
		return errors.Annotatef(err, "serialize")
	}

	err = a.pubRedis.Publish(channel, string(data)).Err()

	return errors.Annotate(err, "publish")
}

func (a *RedisAdapter) Broadcast(msg Message) error {
	channel := a.keys.roomChannel
	a.log.Printf("RedisAdapter.Broadcast type: %s to %s", msg.Type, channel)

	err := a.publish(channel, msg)

	return errors.Annotate(err, "broadcast")
}

func (a *RedisAdapter) localBroadcast(clients map[string]ClientWriter, msg Message) (err error) {
	a.log.Printf("RedisAdapter.localBroadcast in room %s of message type: %s", a.room, msg.Type)

	var errs MultiErrorHandler

	for _, client := range clients {
		if err := a.localEmit(client, msg); err != nil {
			errs.Add(errors.Trace(err))
		}
	}

	return errors.Trace(errs.Err())
}

func (a *RedisAdapter) Emit(clientID string, msg Message) error {
	channel := getClientChannelName(a.prefix, a.room, clientID)
	a.log.Printf("Emit clientID: %s, type: %s to %s", clientID, msg.Type, channel)

	data, err := a.serializer.Serialize(msg)
	if err != nil {
		return errors.Annotatef(err, "serialize message")
	}

	err = a.pubRedis.Publish(channel, string(data)).Err()
	return errors.Annotatef(err, "publish message")
}

func (a *RedisAdapter) localEmit(client ClientWriter, msg Message) error {
	clientID := client.ID()

	a.log.Printf("RedisAdapter.localEmit clientID: %s, type: %s", clientID, msg.Type)

	err := client.Write(msg)
	if err != nil {
		return errors.Annotatef(err, "write %s %s", a.room, clientID)
	}

	return nil
}
