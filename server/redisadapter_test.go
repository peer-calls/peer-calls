package server_test

import (
	"context"
	pkgErrors "errors"
	"sync"
	"testing"

	"time"

	"github.com/go-redis/redis/v7"
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/message"
	"github.com/peer-calls/peer-calls/v4/server/test"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"nhooyr.io/websocket"
)

func errIs(err error, target error) bool {
	return pkgErrors.Is(err, target)
}

func configureRedis(t *testing.T) (*redis.Client, *redis.Client, func()) {
	subRedis := redis.NewClient(&redis.Options{
		Addr:        "localhost:6379",
		DialTimeout: 10 * time.Second,
	})
	pubRedis := redis.NewClient(&redis.Options{
		Addr:        "localhost:6379",
		DialTimeout: 10 * time.Second,
	})
	return pubRedis, subRedis, func() {
		pubRedis.Close()
		subRedis.Close()
	}
}

func getClientIDs(t *testing.T, a *server.RedisAdapter) map[identifiers.ClientID]string {
	clientIDs, err := a.Clients()
	assert.Nil(t, err)
	return clientIDs
}

func TestRedisAdapter_add_remove_client(t *testing.T) {
	defer goleak.VerifyNone(t)
	pub, sub, stop := configureRedis(t)
	defer stop()
	adapter1 := server.NewRedisAdapter(test.NewLogger(), pub, sub, "peercalls", room)
	mockWriter1 := NewMockWriter()
	defer close(mockWriter1.out)

	client1 := server.NewClient(mockWriter1)
	client1.SetMetadata("a")
	mockWriter2 := NewMockWriter()
	defer close(mockWriter2.out)

	client2 := server.NewClient(mockWriter2)
	client2.SetMetadata("b")

	var wg sync.WaitGroup
	wg.Add(2)

	for _, client := range []*server.Client{client1, client2} {
		go func(client *server.Client) {
			for range client.Messages() {
			}
			err := client.Err()
			assert.True(t, errIs(errors.Cause(err), context.Canceled), "expected error to be context.Canceled, but was: %s", err)
			wg.Done()
		}(client)
	}

	assert.Nil(t, adapter1.Add(client1))
	t.Log("waiting for room join message broadcast (1)")

	recv := func(t *testing.T, ch <-chan []byte) []byte {
		t.Helper()

		select {
		case msg := <-ch:
			return msg
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for message")
			return nil
		}
	}

	assert.Equal(t, serialize(t, message.NewRoomJoin(room, message.RoomJoin{ClientID: client1.ID(), Metadata: "a"})), recv(t, mockWriter1.out))

	adapter2 := server.NewRedisAdapter(test.NewLogger(), pub, sub, "peercalls", room)
	assert.Nil(t, adapter2.Add(client2))
	t.Log("waiting for room join message broadcast (2)")
	assert.Equal(t, serialize(t, message.NewRoomJoin(room, message.RoomJoin{ClientID: client2.ID(), Metadata: "b"})), recv(t, mockWriter1.out))
	assert.Equal(t, serialize(t, message.NewRoomJoin(room, message.RoomJoin{ClientID: client2.ID(), Metadata: "b"})), recv(t, mockWriter2.out))
	assert.Equal(t, map[identifiers.ClientID]string{client1.ID(): "a", client2.ID(): "b"}, getClientIDs(t, adapter1))
	assert.Equal(t, map[identifiers.ClientID]string{client1.ID(): "a", client2.ID(): "b"}, getClientIDs(t, adapter2))

	assert.True(t, adapter1.SetMetadata(client1.ID(), "aaa"))
	assert.True(t, adapter2.SetMetadata(client2.ID(), "bbb"))
	metadata, ok := adapter1.Metadata(client1.ID())
	assert.True(t, ok)
	assert.Equal(t, "aaa", metadata)
	metadata, ok = adapter2.Metadata(client1.ID())
	assert.True(t, ok)
	assert.Equal(t, "aaa", metadata)
	metadata, ok = adapter1.Metadata(client2.ID())
	assert.True(t, ok)
	assert.Equal(t, "bbb", metadata)
	metadata, ok = adapter2.Metadata(client2.ID())
	assert.True(t, ok)
	assert.Equal(t, "bbb", metadata)

	assert.Nil(t, adapter1.Remove(client1.ID()))
	t.Log("waiting for client id removal", client1.ID())
	leaveMessage, err := serializer.Deserialize(recv(t, mockWriter2.out))
	assert.Nil(t, err)

	assert.Equal(t, room, leaveMessage.Room)
	assert.Equal(t, message.TypeRoomLeave, leaveMessage.Type)
	assert.Equal(t, client1.ID(), leaveMessage.Payload.RoomLeave)
	// FIXME strong types.
	// assert.Equal(t, server.NewMessageRoomLeave(room, client1.ID()), leaveMessage)
	assert.Equal(t, map[identifiers.ClientID]string{client2.ID(): "bbb"}, getClientIDs(t, adapter2))

	assert.Nil(t, adapter2.Remove(client2.ID()))
	assert.Equal(t, map[identifiers.ClientID]string{}, getClientIDs(t, adapter2))

	t.Log("stopping...")
	for _, stop := range []func() error{adapter1.Close, adapter2.Close} {
		err := stop()
		assert.Equal(t, nil, err)
	}

	err = client1.Close(websocket.StatusNormalClosure, "")
	assert.NoError(t, err, "closing websocket client1")

	err = client2.Close(websocket.StatusNormalClosure, "")
	assert.NoError(t, err, "closing websocket client2")

	wg.Wait()
}
