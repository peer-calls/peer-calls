package server

import (
	"context"
	"fmt"
	"sync"
	"time"

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
	id         string
	conn       WSReadWriter
	metadata   string
	serializer ByteSerializer
	onceClose  sync.Once

	errMu sync.RWMutex
	err   error
}

// Creates a new websocket client.
func NewClient(conn WSReadWriter) *Client {
	return NewClientWithID(conn, "")
}

func NewClientWithID(conn WSReadWriter, id string) *Client {
	if id == "" {
		id = NewUUIDBase62()
	}
	return &Client{
		id:   id,
		conn: conn,
	}
}

func (c *Client) SetMetadata(metadata string) {
	c.metadata = metadata
}

func (c *Client) Metadata() string {
	return c.metadata
}

// Writes a message to websocket with timeout.
func (c *Client) WriteTimeout(ctx context.Context, timeout time.Duration, msg Message) error {
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

// Write writes a message to client socket
func (c *Client) Write(msg Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err := c.WriteTimeout(ctx, 5*time.Second, msg)
	if err != nil {
		return fmt.Errorf("client.Write: %w", err)
	}
	return nil
}

func (c *Client) read(ctx context.Context) (message Message, err error) {
	typ, data, err := c.conn.Read(ctx)
	if err != nil {
		err = fmt.Errorf("client.read - error reading data: %w", err)
		return
	}
	message, err = c.serializer.Deserialize(data)
	if err != nil {
		err = fmt.Errorf("client.read - error deserializing data: %w", err)
		return
	}
	if typ != websocket.MessageText {
		err = fmt.Errorf("client.read - expected text message: %w", err)
	}
	return
}

func (c *Client) Err() error {
	c.errMu.RLock()
	defer c.errMu.RUnlock()
	return c.err
}

func (c *Client) Subscribe(ctx context.Context) <-chan Message {
	msgChan := make(chan Message)

	go func() {
		for {
			message, err := c.read(ctx)
			if err != nil {
				c.errMu.Lock()
				close(msgChan)
				c.err = err
				c.errMu.Unlock()
				return
			}
			msgChan <- message
		}
	}()

	return msgChan
}
