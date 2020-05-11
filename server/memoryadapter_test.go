package server_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/peer-calls/peer-calls/server"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMemoryAdapter_add_remove_clients(t *testing.T) {
	goleak.VerifyNone(t)
	adapter := server.NewMemoryAdapter(room)
	mockWriter := NewMockWriter()
	client := server.NewClient(mockWriter)
	client.SetMetadata("a")
	clientID := client.ID()
	err := adapter.Add(client)
	assert.Nil(t, err)
	clientIDs, err := adapter.Clients()
	assert.Nil(t, err)
	assert.Equal(t, map[string]string{clientID: "a"}, clientIDs)
	size, err := adapter.Size()
	assert.Nil(t, err)
	assert.Equal(t, 1, size)
	adapter.Remove(clientID)
	clientIDs, err = adapter.Clients()
	assert.Nil(t, err)
	assert.Equal(t, map[string]string{}, clientIDs)
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
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		msgChan := client.Subscribe(ctx)
		for range msgChan {
		}
		err := client.Err()
		assert.True(t, errors.Is(err, context.Canceled), "expected context.Canceled, but got: %s", err)
		wg.Done()
	}()
	msg := server.NewMessage("test-type", room, []byte("test"))
	adapter.Emit(client.ID(), msg)
	msg1 := <-mockWriter.out
	joinMessage := serialize(t, server.NewMessageRoomJoin(room, client.ID(), client.Metadata()))
	assert.Equal(t, joinMessage, msg1)
	msg2 := <-mockWriter.out
	cancel()
	assert.Equal(t, serialize(t, msg), msg2)
	wg.Wait()
}

func TestMemoryAdapter_emitMissing(t *testing.T) {
	goleak.VerifyNone(t)
	adapter := server.NewMemoryAdapter(room)
	msg := server.NewMessage("test-type", room, []byte("test"))
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
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		msgChan := client1.Subscribe(ctx)
		for range msgChan {
		}
		err := client1.Err()
		assert.True(t, errors.Is(err, context.Canceled), "expected context.Canceled, but got: %s", err)
		wg.Done()
	}()
	go func() {
		msgChan := client2.Subscribe(ctx)
		for range msgChan {
		}
		err := client2.Err()
		assert.True(t, errors.Is(err, context.Canceled), "expected context.Canceled, but got: %s", err)
		wg.Done()
	}()
	msg := server.NewMessage("test-type", room, []byte("test"))
	adapter.Broadcast(msg)
	assert.Equal(t, serialize(t, server.NewMessageRoomJoin(room, client1.ID(), "")), <-mockWriter1.out)
	assert.Equal(t, serialize(t, server.NewMessageRoomJoin(room, client2.ID(), "")), <-mockWriter1.out)
	assert.Equal(t, serialize(t, server.NewMessageRoomJoin(room, client2.ID(), "")), <-mockWriter2.out)
	serializedMsg := serialize(t, msg)
	assert.Equal(t, serializedMsg, <-mockWriter1.out)
	assert.Equal(t, serializedMsg, <-mockWriter2.out)
	cancel()
	wg.Wait()
}
