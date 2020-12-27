package server_test

import (
	"context"
	"testing"

	"github.com/peer-calls/peer-calls/server"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

// This package contains commonly used test variables

const room = "test-room"

var serializer server.ByteSerializer

type MockWSWriter struct {
	out chan []byte
}

func NewMockWriter() *MockWSWriter {
	return &MockWSWriter{
		out: make(chan []byte, 16),
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

func serialize(t *testing.T, msg server.Message) []byte {
	t.Helper()
	data, err := serializer.Serialize(msg)
	require.Nil(t, err)
	return data
}
