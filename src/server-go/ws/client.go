package ws

import (
	"context"
	"time"

	"github.com/google/uuid"
	"nhooyr.io/websocket"
)

type WSWriter interface {
	Write(ctx context.Context, typ websocket.MessageType, msg []byte) error
}

// An abstraction for sending messages to websocket using channels.
type Client struct {
	id       string
	conn     WSWriter
	messages chan []byte
}

// Creates a new websocket client.
func NewClient(conn WSWriter) *Client {
	return &Client{
		id:       uuid.New().String(),
		conn:     conn,
		messages: make(chan []byte, 16),
	}
}

// Writes a message to websocket with timeout.
func (c *Client) WriteTimeout(ctx context.Context, timeout time.Duration, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.conn.Write(ctx, websocket.MessageText, msg)
}

func (c *Client) ID() string {
	return c.id
}

// Gets the channel to write messages to. Messages sent here will be written
// to the websocket and received by the other side.
func (c *Client) Messages() chan<- []byte {
	return c.messages
}

// Subscribes to messages and writes messages to the websocket. This method
// blocks until the channel is closed, or the context is done. Should be
// called from the HTTP handler method.
func (c *Client) Subscribe(ctx context.Context) error {
	for {
		select {
		case msg := <-c.messages:
			err := c.WriteTimeout(ctx, time.Second*5, msg)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
