package wsmemory_test

import (
	"context"
	"sync"
	"testing"

	"github.com/jeremija/peer-calls/src/server-go/ws"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsmemory"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsmessage"
	"github.com/stretchr/testify/assert"
	"nhooyr.io/websocket"
)

const room = "test-room"

var serializer wsmessage.ByteSerializer

type MockWSWriter struct {
	out chan []byte
}

func NewMockWriter() *MockWSWriter {
	return &MockWSWriter{
		out: make(chan []byte),
	}
}

func (w *MockWSWriter) Write(ctx context.Context, typ websocket.MessageType, msg []byte) error {
	w.out <- msg
	return nil
}

func (w *MockWSWriter) Read(ctx context.Context) (typ websocket.MessageType, msg []byte, err error) {
	<-ctx.Done()
	err = ctx.Err()
	return
}

func TestMemoryAdapter_add_remove_clients(t *testing.T) {
	adapter := wsmemory.NewMemoryAdapter(room)
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
	adapter := wsmemory.NewMemoryAdapter(room)
	mockWriter := NewMockWriter()
	defer close(mockWriter.out)
	client := ws.NewClient(mockWriter)
	adapter.Add(client)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err := client.Subscribe(ctx, func(msg wsmessage.Message) {})
		assert.Equal(t, context.Canceled, err)
		wg.Done()
	}()
	msg := wsmessage.NewMessage("test-type", room, []byte("test"))
	adapter.Emit(client.ID(), msg)
	msg1 := <-mockWriter.out
	joinMessage := serializer.Serialize(wsmessage.NewMessageRoomJoin(room, client.ID()))
	assert.Equal(t, joinMessage, msg1)
	msg2 := <-mockWriter.out
	cancel()
	assert.Equal(t, serializer.Serialize(msg), msg2)
	wg.Wait()
}

func TestMemoryAdapter_emitMissing(t *testing.T) {
	adapter := wsmemory.NewMemoryAdapter(room)
	msg := wsmessage.NewMessage("test-type", room, []byte("test"))
	adapter.Emit("123", msg)
}

func TestMemoryAdapter_Brodacast(t *testing.T) {
	adapter := wsmemory.NewMemoryAdapter(room)
	mockWriter1 := NewMockWriter()
	client1 := ws.NewClient(mockWriter1)
	mockWriter2 := NewMockWriter()
	client2 := ws.NewClient(mockWriter2)
	defer close(mockWriter1.out)
	defer close(mockWriter2.out)
	adapter.Add(client1)
	adapter.Add(client2)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		err := client1.Subscribe(ctx, func(msg wsmessage.Message) {})
		assert.Equal(t, context.Canceled, err)
		wg.Done()
	}()
	go func() {
		err := client2.Subscribe(ctx, func(msg wsmessage.Message) {})
		assert.Equal(t, context.Canceled, err)
		wg.Done()
	}()
	msg := wsmessage.NewMessage("test-type", room, []byte("test"))
	adapter.Broadcast(msg)
	assert.Equal(t, serializer.Serialize(wsmessage.NewMessageRoomJoin(room, client1.ID())), <-mockWriter1.out)
	assert.Equal(t, serializer.Serialize(wsmessage.NewMessageRoomJoin(room, client2.ID())), <-mockWriter1.out)
	assert.Equal(t, serializer.Serialize(wsmessage.NewMessageRoomJoin(room, client2.ID())), <-mockWriter2.out)
	serializedMsg := serializer.Serialize(msg)
	assert.Equal(t, serializedMsg, <-mockWriter1.out)
	assert.Equal(t, serializedMsg, <-mockWriter2.out)
	cancel()
	wg.Wait()
}
