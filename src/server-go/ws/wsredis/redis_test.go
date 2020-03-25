package wsredis_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v7"
	"github.com/jeremija/peer-calls/src/server-go/ws"
	"github.com/stretchr/testify/require"
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

func configureRedis(t *testing.T) (*redis.Client, func()) {
	r, err := miniredis.Run()
	require.Nil(t, err)
	redis := redis.NewClient(&redis.Options{
		Addr: r.Addr(),
	})
	return redis, func() {
		r.Close()
		redis.Close()
	}
}

func createClient() *ws.Client {
	mockWriter := NewMockWriter()
	return ws.NewClient(mockWriter)
}

func TestRedisAdapter_add_remove_clients(t *testing.T) {
	// redis, stop := configureRedis(t)
	// defer stop()
	// adapter := wsredis.NewRedisAdapter(redis, "peercalls", "myroom")
	// client1 := createClient()
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()
	// go func() {
	// 	err := adapter.Subscribe(ctx)
	// 	assert.Equal(t, context.Canceled, err)
	// }()
	// adapter.Add(client1)
	// i := 0
	// for {
	// 	i++
	// 	if adapter.Size() == 1 {
	// 		break
	// 	}
	// 	time.Sleep(100 * time.Millisecond)
	// 	require.LessOrEqual(t, t, 10, "waiting timed out")
	// }
}
