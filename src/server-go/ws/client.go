package ws

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsmessage"
	"nhooyr.io/websocket"
)

type WSWriter interface {
	Write(ctx context.Context, typ websocket.MessageType, msg []byte) error
}

// An abstraction for sending out to websocket using channels.
type Client struct {
	id           string
	conn         WSWriter
	writeChannel chan wsmessage.Message
	serializer   wsmessage.ByteSerializer
}

// Creates a new websocket client.
func NewClient(conn WSWriter) *Client {
	return &Client{
		id:           uuid.New().String(),
		conn:         conn,
		writeChannel: make(chan wsmessage.Message, 16),
	}
}

// Writes a message to websocket with timeout.
func (c *Client) WriteTimeout(ctx context.Context, timeout time.Duration, msg wsmessage.Message) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.conn.Write(ctx, websocket.MessageText, c.serializer.Serialize(msg))
}

func (c *Client) ID() string {
	return c.id
}

// Gets the channel to write out to. Messages sent here will be written
// to the websocket and received by the other side.
func (c *Client) WriteChannel() chan<- wsmessage.Message {
	return c.writeChannel
}

// Subscribes to out and writes out to the websocket. This method
// blocks until the channel is closed, or the context is done. Should be
// called from the HTTP handler method.
func (c *Client) Subscribe(ctx context.Context) error {
	for {
		select {
		case msg := <-c.writeChannel:
			err := c.WriteTimeout(ctx, time.Second*5, msg)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
