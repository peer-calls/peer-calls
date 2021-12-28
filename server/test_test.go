package server_test

import (
	"context"
	"os"
	"testing"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/message"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

// This package contains commonly used test variables

const room identifiers.RoomID = "test-room"

// nolint:gochecknoglobals
var serializer server.ByteSerializer

type MockWSWriter struct {
	out      chan []byte
	closeCtx context.Context
	cancel   context.CancelFunc
}

func NewMockWriter() *MockWSWriter {
	closeCtx, cancel := context.WithCancel(context.Background())

	return &MockWSWriter{
		closeCtx: closeCtx,
		cancel:   cancel,
		out:      make(chan []byte, 16),
	}
}

func (w *MockWSWriter) Write(ctx context.Context, typ websocket.MessageType, msg []byte) error {
	w.out <- msg
	return nil
}

func (w *MockWSWriter) Read(ctx context.Context) (typ websocket.MessageType, msg []byte, err error) {
	select {
	case <-ctx.Done():
		err = errors.Trace(ctx.Err())
	case <-w.closeCtx.Done():
		err = errors.Trace(w.closeCtx.Err())
	}

	return
}

func (w *MockWSWriter) Close(statusCode websocket.StatusCode, reason string) error {
	w.cancel()
	return nil
}

func serialize(t *testing.T, msg message.Message) []byte {
	t.Helper()
	data, err := serializer.Serialize(msg)
	require.Nil(t, err)
	return data
}

// nolint:gochecknoglobals
var embed = server.Embed{
	Templates: os.DirFS("templates"),
	Static:    os.DirFS("../build"),
	Resources: os.DirFS("../res"),
}
