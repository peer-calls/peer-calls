package wsredis_test

import (
	"context"
	"sort"
	"sync"
	"testing"

	"github.com/go-redis/redis/v7"
	"github.com/jeremija/peer-calls/src/server-go/ws"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsmessage"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsredis"
	"github.com/stretchr/testify/assert"
	"nhooyr.io/websocket"
)

const room = "myroom"

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

func configureRedis(t *testing.T) (*redis.Client, *redis.Client, func()) {
	subRedis := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	pubRedis := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	return pubRedis, subRedis, func() {
		pubRedis.Close()
		subRedis.Close()
	}
}

func assertEqualSorted(t *testing.T, s1 []string, s2 []string) {
	t.Helper()
	sort.Strings(s1)
	sort.Strings(s2)
	assert.Equal(t, s1, s2)
}

func TestRedisAdapter_add_remove_client(t *testing.T) {
	pub, sub, stop := configureRedis(t)
	defer stop()
	adapter1 := wsredis.NewRedisAdapter(pub, sub, "peercalls", room)
	adapter2 := wsredis.NewRedisAdapter(pub, sub, "peercalls", room)
	mockWriter1 := NewMockWriter()
	defer close(mockWriter1.out)
	client1 := ws.NewClient(mockWriter1)
	mockWriter2 := NewMockWriter()
	defer close(mockWriter2.out)
	client2 := ws.NewClient(mockWriter2)
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(2)

	stop1 := adapter1.Subscribe()
	stop2 := adapter2.Subscribe()
	for _, client := range []*ws.Client{client1, client2} {
		go func(client *ws.Client) {
			err, _ := client.Subscribe(ctx)
			assert.Equal(t, context.Canceled, err)
			wg.Done()
		}(client)
	}

	adapter1.Add(client1)
	t.Log("waiting for room join message broadcast (1)")
	assert.Equal(t, serializer.Serialize(wsmessage.NewMessageRoomJoin(room, client1.ID())), <-mockWriter1.out)

	adapter2.Add(client2)
	t.Log("waiting for room join message broadcast (2)")
	assert.Equal(t, serializer.Serialize(wsmessage.NewMessageRoomJoin(room, client2.ID())), <-mockWriter1.out)
	assert.Equal(t, serializer.Serialize(wsmessage.NewMessageRoomJoin(room, client2.ID())), <-mockWriter2.out)
	assertEqualSorted(t, []string{client1.ID(), client2.ID()}, adapter1.Clients())
	assertEqualSorted(t, []string{client1.ID(), client2.ID()}, adapter2.Clients())

	adapter1.Remove(client1.ID())
	t.Log("waiting for client id removal", client1.ID())
	leaveMessage := serializer.Deserialize(<-mockWriter2.out)
	assert.Equal(t, wsmessage.NewMessageRoomLeave(room, client1.ID()), leaveMessage)
	assert.Equal(t, []string{client2.ID()}, adapter2.Clients())

	adapter2.Remove(client2.ID())
	assert.Equal(t, make([]string, 0), adapter2.Clients())

	t.Log("stopping...")
	for _, stop := range []func() error{stop1, stop2} {
		err := stop()
		assert.Equal(t, context.Canceled, err)
	}
	cancel()
	wg.Wait()
}
