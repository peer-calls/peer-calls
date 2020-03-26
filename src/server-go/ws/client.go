package ws

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsmessage"
	"nhooyr.io/websocket"
)

type WSWriter interface {
	Write(ctx context.Context, typ websocket.MessageType, msg []byte) error
}

type WSReader interface {
	Read(ctx context.Context) (websocket.MessageType, []byte, error)
}

type WSReadWriter interface {
	WSReader
	WSWriter
}

// An abstraction for sending out to websocket using channels.
type Client struct {
	id           string
	conn         WSReadWriter
	writeChannel chan wsmessage.Message
	readChannel  chan wsmessage.Message
	serializer   wsmessage.ByteSerializer
}

// Creates a new websocket client.
func NewClient(conn WSReadWriter) *Client {
	return &Client{
		id:           uuid.New().String(),
		conn:         conn,
		writeChannel: make(chan wsmessage.Message, 16),
		readChannel:  make(chan wsmessage.Message, 16),
	}
}

// Writes a message to websocket with timeout.
func (c *Client) WriteTimeout(ctx context.Context, timeout time.Duration, msg wsmessage.Message) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	data, err := c.serializer.Serialize(msg)
	if err != nil {
		return fmt.Errorf("client.WriteTimeout - error serializing message: %w", err)
	}
	return c.conn.Write(ctx, websocket.MessageText, data)
}

func (c *Client) ID() string {
	return c.id
}

// Gets the channel to write out to. Messages sent here will be written
// to the websocket and received by the other side.
func (c *Client) WriteChannel() chan<- wsmessage.Message {
	return c.writeChannel
}

// Subscribes
func (c *Client) subscribeRead(ctx context.Context) error {
	for {
		typ, data, err := c.conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("client.subscribeRead - error reading data: %w", err)
		}
		message, err := c.serializer.Deserialize(data)
		if err != nil {
			return fmt.Errorf("client.subscribeRead - error deserializing data: %w", err)
		}
		if typ == websocket.MessageText {
			c.readChannel <- message
		}
	}
}

func (c *Client) Subscribe(ctx context.Context, handle func(wsmessage.Message)) error {
	ctx, cancel := context.WithCancel(ctx)

	defer func() {
		cancel()
		close(c.readChannel)
		close(c.writeChannel)
	}()

	readErr := make(chan error)
	go func() {
		readErr <- c.subscribeRead(ctx)
		close(readErr)
	}()

	for {
		select {
		case msg := <-c.writeChannel:
			err := c.WriteTimeout(ctx, time.Second*5, msg)
			if err != nil {
				return err
			}
		case msg := <-c.readChannel:
			handle(msg)
		case err := <-readErr:
			return err
		case <-ctx.Done():
			err := ctx.Err()
			return err
		}
	}
}
