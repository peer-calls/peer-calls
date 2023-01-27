package server_test

import (
	"context"
	"sync"
	"testing"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/message"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"nhooyr.io/websocket"
)

func TestMemoryAdapter_add_remove_clients(t *testing.T) {
	goleak.VerifyNone(t)
	adapter := server.NewMemoryAdapter(room)
	mockWriter := NewMockWriter()
	client := server.NewClient(mockWriter)

	defer client.Close(websocket.StatusNormalClosure, "")

	client.SetMetadata("a")
	clientID := client.ID()

	err := adapter.Add(client)
	assert.Nil(t, err)

	clientIDs, err := adapter.Clients()
	assert.Nil(t, err)
	assert.Equal(t, map[identifiers.ClientID]string{clientID: "a"}, clientIDs)

	size, err := adapter.Size()
	assert.Nil(t, err)
	assert.Equal(t, 1, size)

	err = adapter.Remove(clientID)
	assert.Nil(t, err)
	clientIDs, err = adapter.Clients()

	assert.Nil(t, err)
	assert.Equal(t, map[identifiers.ClientID]string{}, clientIDs)

	size, err = adapter.Size()
	assert.Nil(t, err)
	assert.Equal(t, 0, size)
}

func TestMemoryAdapter_emitFound(t *testing.T) {
	goleak.VerifyNone(t)
	adapter := server.NewMemoryAdapter(room)
	mockWriter := NewMockWriter()
	defer close(mockWriter.out)
	client := server.NewClient(mockWriter)
	adapter.Add(client)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		msgChan := client.Messages()
		for range msgChan {
		}
		err := client.Err()
		assert.True(t, errIs(errors.Cause(err), context.Canceled), "expected context.Canceled, but got: %s", err)
		wg.Done()
	}()

	msg := message.NewReady(room, message.Ready{
		Nickname: "test",
	})
	adapter.Emit(client.ID(), msg)
	msg1 := <-mockWriter.out

	joinMessage := serialize(t, message.NewRoomJoin(room, message.RoomJoin{
		ClientID: client.ID(),
		Metadata: client.Metadata(),
	}))

	assert.Equal(t, joinMessage, msg1)
	msg2 := <-mockWriter.out

	client.Close(websocket.StatusNormalClosure, "")

	assert.Equal(t, serialize(t, msg), msg2)
	wg.Wait()
}

func TestMemoryAdapter_emitMissing(t *testing.T) {
	goleak.VerifyNone(t)
	adapter := server.NewMemoryAdapter(room)

	msg := message.NewReady(room, message.Ready{
		Nickname: "test",
	})

	adapter.Emit("123", msg)
}

func TestMemoryAdapter_Broadcast(t *testing.T) {
	goleak.VerifyNone(t)
	adapter := server.NewMemoryAdapter(room)
	mockWriter1 := NewMockWriter()
	client1 := server.NewClient(mockWriter1)
	mockWriter2 := NewMockWriter()
	client2 := server.NewClient(mockWriter2)

	defer close(mockWriter1.out)
	defer close(mockWriter2.out)

	assert.Nil(t, adapter.Add(client1))
	assert.Nil(t, adapter.Add(client2))

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		for range client1.Messages() {
		}
		err := client1.Err()
		assert.True(t, errIs(errors.Cause(err), context.Canceled), "expected context.Canceled, but got: %s", err)
		wg.Done()
	}()
	go func() {
		for range client2.Messages() {
		}
		err := client2.Err()
		assert.True(t, errIs(errors.Cause(err), context.Canceled), "expected context.Canceled, but got: %s", err)
		wg.Done()
	}()

	msg := message.NewReady(room, message.Ready{
		Nickname: "test",
	})
	adapter.Broadcast(msg)

	assert.Equal(t, serialize(t, message.NewRoomJoin(room, message.RoomJoin{ClientID: client1.ID(), Metadata: ""})), <-mockWriter1.out)
	assert.Equal(t, serialize(t, message.NewRoomJoin(room, message.RoomJoin{ClientID: client2.ID(), Metadata: ""})), <-mockWriter1.out)
	assert.Equal(t, serialize(t, message.NewRoomJoin(room, message.RoomJoin{ClientID: client2.ID(), Metadata: ""})), <-mockWriter2.out)

	serializedMsg := serialize(t, msg)
	assert.Equal(t, serializedMsg, <-mockWriter1.out)
	assert.Equal(t, serializedMsg, <-mockWriter2.out)

	err := client1.Close(websocket.StatusNormalClosure, "")
	assert.NoError(t, err, "closing websocket client1")

	err = client2.Close(websocket.StatusNormalClosure, "")
	assert.NoError(t, err, "closing websocket client2")

	wg.Wait()
}
