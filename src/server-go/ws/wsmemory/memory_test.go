package wsmemory_test

import (
	"context"
	"testing"

	"github.com/jeremija/peer-calls/src/server-go/ws"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsmemory"
	"github.com/stretchr/testify/assert"
	"nhooyr.io/websocket"
)

type MockWSWriter struct {
	messages chan []byte
}

func NewMockWriter() *MockWSWriter {
	return &MockWSWriter{
		messages: make(chan []byte),
	}
}

func (w *MockWSWriter) Write(ctx context.Context, typ websocket.MessageType, msg []byte) error {
	w.messages <- msg
	return nil
}

func TestMemoryAdapter_add_remove_clients(t *testing.T) {
	adapter := wsmemory.NewMemoryAdapter()
	mockWriter := NewMockWriter()
	client := ws.NewClient(mockWriter)
	clientID := client.ID()
	adapter.Add(client)
	assert.Equal(t, []string{clientID}, adapter.Clients())
	assert.Equal(t, 1, adapter.Size())
	adapter.Remove(clientID)
	assert.Equal(t, []string{}, adapter.Clients())
	assert.Equal(t, 0, adapter.Size())
}

func TestMemoryAdapter_emitFound(t *testing.T) {
	adapter := wsmemory.NewMemoryAdapter()
	mockWriter := NewMockWriter()
	defer close(mockWriter.messages)
	client := ws.NewClient(mockWriter)
	adapter.Add(client)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := client.Subscribe(ctx)
		assert.Equal(t, context.Canceled, err)
	}()
	msg := []byte("test")
	adapter.Emit(client.ID(), msg)
	msg2 := <-mockWriter.messages
	cancel()
	assert.Equal(t, msg, msg2)
}

func TestMemoryAdapter_emitMissing(t *testing.T) {
	adapter := wsmemory.NewMemoryAdapter()
	adapter.Emit("123", []byte("test"))
}

func TestMemoryAdapter_Brodacst(t *testing.T) {
	adapter := wsmemory.NewMemoryAdapter()
	mockWriter1 := NewMockWriter()
	client1 := ws.NewClient(mockWriter1)
	mockWriter2 := NewMockWriter()
	client2 := ws.NewClient(mockWriter2)
	defer close(mockWriter1.messages)
	defer close(mockWriter2.messages)
	adapter.Add(client1)
	adapter.Add(client2)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := client1.Subscribe(ctx)
		assert.Equal(t, context.Canceled, err)
	}()
	go func() {
		err := client2.Subscribe(ctx)
		assert.Equal(t, context.Canceled, err)
	}()
	msg := []byte("test")
	adapter.Broadcast(msg)
	msg1 := <-mockWriter1.messages
	msg2 := <-mockWriter2.messages
	cancel()
	assert.Equal(t, msg, msg1)
	assert.Equal(t, msg, msg2)
}
