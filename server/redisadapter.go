package server

import (
	"context"
	e "errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/message"
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
	log          logger.Logger
	serializer   Serializer
	deserializer Deserializer

	clientsMu *sync.RWMutex
	// contains local clients connected to current instance
	clients map[identifiers.ClientID]ClientWriter
	// contains IDs of all clients in room, including those from other instances
	prefix   string
	room     identifiers.RoomID
	pubRedis *redis.Client // TODO replace this with interface
	subRedis *redis.Client
	keys     struct {
		roomChannel   string
		roomClients   string
		clientPattern string
	}
	stop func() error
}

func getRoomChannelName(prefix string, room identifiers.RoomID) string {
	// TODO escape room name, what if it has ":" in the name?
	return prefix + ":room:" + room.String() + ":broadcast"
}

func getClientChannelName(prefix string, room identifiers.RoomID, clientID identifiers.ClientID) string {
	// TODO escape room name, what if it has ":" in the name?
	return prefix + ":room:" + room.String() + ":client:" + clientID.String()
}

func getRoomClientsName(prefix string, room identifiers.RoomID) string {
	// TODO escape room name, what if it has ":" in the name?
	return prefix + ":room:" + room.String() + ":clients"
}

func NewRedisAdapter(
	log logger.Logger,
	pubRedis *redis.Client,
	subRedis *redis.Client,
	prefix string,
	room identifiers.RoomID,
) *RedisAdapter {
	var (
		clientsMu      sync.RWMutex
		byteSerializer ByteSerializer
	)

	adapter := RedisAdapter{
		log: log.WithNamespaceAppended("redis_adapter").WithCtx(logger.Ctx{
			"room_id": room,
		}),
		serializer:   byteSerializer,
		deserializer: byteSerializer,
		clients:      map[identifiers.ClientID]ClientWriter{},
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

	a.log.Trace("Add", logger.Ctx{
		"client_id": clientID,
	})

	a.clientsMu.Lock()

	// TODO what if a client with the same ID has joined another
	// node? We need to be smarter about this.
	//
	// Perhaps an easy solution is to add a node ID as a key prefix.
	if _, ok := a.clients[clientID]; ok {
		err = errors.Annotatef(ErrDuplicateClientID, "%s", clientID)
	} else {
		a.clients[clientID] = client
	}

	a.clientsMu.Unlock()

	if err != nil {
		return errors.Trace(err)
	}

	join := message.RoomJoin{
		ClientID: clientID,
		Metadata: client.Metadata(),
	}

	err = a.Broadcast(message.NewRoomJoin(a.room, join))
	if err != nil {
		return errors.Annotatef(err, "clientID: %s", clientID)
	}

	return nil
}

func (a *RedisAdapter) Remove(clientID identifiers.ClientID) error {
	a.log.Trace("Remove", logger.Ctx{
		"client_id": clientID,
	})

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

func (a *RedisAdapter) removeAll(clientIDs []identifiers.ClientID) (err error) {
	var errs MultiErrorHandler

	for _, clientID := range clientIDs {
		if err := a.remove(clientID); err != nil {
			errs.Add(errors.Trace(err))
		}
	}

	return errors.Trace(errs.Err())
}

func (a *RedisAdapter) remove(clientID identifiers.ClientID) (err error) {
	var errs MultiErrorHandler

	// can only remove clients connected to this adapter
	if err = a.pubRedis.HDel(a.keys.roomClients, clientID.String()).Err(); err != nil {
		errs.Add(errors.Annotatef(err, "hdel %s %s", a.keys.roomClients, clientID))
	}

	if err = a.Broadcast(message.NewRoomLeave(a.room, clientID)); err != nil {
		errs.Add(errors.Annotatef(err, "broadcast room leave %s %s", a.keys.roomClients, clientID))
	}

	return errors.Trace(errs.Err())
}

func (a *RedisAdapter) Metadata(clientID identifiers.ClientID) (metadata string, ok bool) {
	metadata, err := a.pubRedis.HGet(a.keys.roomClients, clientID.String()).Result()

	return metadata, err == nil
}

func (a *RedisAdapter) SetMetadata(clientID identifiers.ClientID, metadata string) (ok bool) {
	logCtx := logger.Ctx{
		"client_id": clientID,
		"metadata":  metadata,
	}

	a.log.Trace("SetMetadata", logCtx)

	_, err := a.pubRedis.HSet(a.keys.roomClients, clientID.String(), metadata).Result()
	if err != nil {
		// FIXME return error
		a.log.Error("SetMetadata", errors.Trace(err), logCtx)
	}

	return err == nil
}

// Returns IDs of all known clients connected to this room
func (a *RedisAdapter) Clients() (map[identifiers.ClientID]string, error) {
	a.log.Trace("Clients", nil)

	r := a.pubRedis.HGetAll(a.keys.roomClients)

	allClients, err := r.Result()
	if err != nil {
		return nil, errors.Annotatef(err, "clients in room: %s", a.room)
	}

	ret := make(map[identifiers.ClientID]string, len(allClients))

	for clientID, metadata := range allClients {
		ret[identifiers.ClientID(clientID)] = metadata
	}

	return ret, nil
}

// Returns count of all known clients connected to this room
func (a *RedisAdapter) Size() (size int, err error) {
	a.log.Trace("Size", nil)

	c, err := a.Clients()

	return len(c), errors.Annotate(err, "size")
}

func (a *RedisAdapter) handleMessage(
	pattern string,
	channel string,
	messageStr string,
) error {
	msg, err := a.deserializer.Deserialize([]byte(messageStr))
	if err != nil {
		return errors.Annotate(err, "deserialize redis subscription")
	}

	handleRoomJoin := func(roomID identifiers.RoomID, join message.RoomJoin) error {
		a.clientsMu.RLock()
		clients := a.localClients()
		a.clientsMu.RUnlock()

		// payload, ok := msg.Payload.(map[string]interface{})
		// if !ok {
		// 	return errors.Errorf("room join: expected a map[string]interface{}, but got payload of type %T", msg.Payload)
		// }

		err = a.pubRedis.HSet(a.keys.roomClients, join.ClientID.String(), join.Metadata).Err()
		if err != nil {
			return errors.Annotate(err, "room join")
		}

		err = a.localBroadcast(clients, msg)
		return errors.Annotate(err, "room join")
	}

	handleRoomLeave := func(roomID identifiers.RoomID, clientID identifiers.ClientID) error {
		a.clientsMu.RLock()
		clients := a.localClients()
		a.clientsMu.RUnlock()

		err = a.localBroadcast(clients, msg)
		if err != nil {
			return errors.Trace(err)
		}

		err = a.pubRedis.HDel(a.keys.roomClients, clientID.String()).Err()

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
		clientID := identifiers.ClientID(params[len(params)-1])

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
		case message.TypeRoomJoin:
			err = errors.Trace(handleRoomJoin(msg.Room, *msg.Payload.RoomJoin))
		case message.TypeRoomLeave:
			err = errors.Trace(handleRoomLeave(msg.Room, msg.Payload.RoomLeave))
		default:
			err = errors.Trace(handleRoomBroadcast())
		}
	case pattern == a.keys.clientPattern:
		err = errors.Trace(handlePattern())
	}

	return errors.Trace(err)
}

// Reads from subscribed keys and dispatches relevant messages to
// client websockets. This method blocks until the context is closed.
func (a *RedisAdapter) subscribe(ctx context.Context, ready chan<- struct{}) error {
	log := a.log.WithCtx(logger.Ctx{
		"room_channel":   a.keys.roomChannel,
		"client_pattern": a.keys.clientPattern,
	})

	log.Trace("subscribe", nil)
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
					log.Error("Handle message", errors.Trace(err), nil)
				}
			}
		case <-ctx.Done():
			err := ctx.Err()
			log.Trace(fmt.Sprintf("Subscribe done: %+v", errors.Trace(err)), nil)
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
	clientIDs := make([]identifiers.ClientID, 0, len(a.clients))

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

func (a *RedisAdapter) localClients() map[identifiers.ClientID]ClientWriter {
	clients := make(map[identifiers.ClientID]ClientWriter, len(a.clients))

	for k, v := range a.clients {
		clients[k] = v
	}

	return clients
}

func (a *RedisAdapter) publish(channel string, msg message.Message) error {
	data, err := a.serializer.Serialize(msg)
	if err != nil {
		return errors.Annotatef(err, "serialize")
	}

	err = a.pubRedis.Publish(channel, string(data)).Err()

	return errors.Annotate(err, "publish")
}

func (a *RedisAdapter) Broadcast(msg message.Message) error {
	channel := a.keys.roomChannel

	a.log.Trace("Broadcast", logger.Ctx{
		"message_type": msg.Type,
		"room_channel": channel,
	})

	err := a.publish(channel, msg)

	return errors.Annotate(err, "broadcast")
}

func (a *RedisAdapter) localBroadcast(clients map[identifiers.ClientID]ClientWriter, msg message.Message) (err error) {
	a.log.Trace("localBroadcast", logger.Ctx{
		"message_type": msg.Type,
	})

	var errs MultiErrorHandler

	for _, client := range clients {
		if err := a.localEmit(client, msg); err != nil {
			errs.Add(errors.Trace(err))
		}
	}

	return errors.Trace(errs.Err())
}

func (a *RedisAdapter) Emit(clientID identifiers.ClientID, msg message.Message) error {
	channel := getClientChannelName(a.prefix, a.room, clientID)

	a.log.Trace("Emit", logger.Ctx{
		"message_type":   msg.Type,
		"client_channel": channel,
	})

	data, err := a.serializer.Serialize(msg)
	if err != nil {
		return errors.Annotatef(err, "serialize message")
	}

	err = a.pubRedis.Publish(channel, string(data)).Err()
	return errors.Annotatef(err, "publish message")
}

func (a *RedisAdapter) localEmit(client ClientWriter, msg message.Message) error {
	clientID := client.ID()

	a.log.Trace("localEmit", logger.Ctx{
		"message_type": msg.Type,
	})

	err := client.Write(msg)
	if err != nil {
		return errors.Annotatef(err, "write %s %s", a.room, clientID)
	}

	return nil
}
