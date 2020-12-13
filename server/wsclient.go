package server

import (
	"context"
	"sync"
	"time"

	"github.com/juju/errors"
	"nhooyr.io/websocket"
)

const defaultWSTimeout = 5 * time.Second

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
		return errors.Annotate(err, "serialize")
	}

	err = c.conn.Write(ctx, websocket.MessageText, data)
	return errors.Annotate(err, "write")
}

func (c *Client) ID() string {
	return c.id
}

// Write writes a message to client socket
func (c *Client) Write(msg Message) error {
	err := c.WriteTimeout(context.Background(), defaultWSTimeout, msg)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (c *Client) read(ctx context.Context) (message Message, err error) {
	typ, data, err := c.conn.Read(ctx)
	if err != nil {
		return Message{}, errors.Annotate(err, "read")
	}

	message, err = c.serializer.Deserialize(data)
	if err != nil {
		return Message{}, errors.Annotate(err, "deserialize")
	}

	if typ != websocket.MessageText {
		return Message{}, errors.Errorf("unexpected text message type, but got %s", typ)
	}

	return message, nil
}

func (c *Client) Err() error {
	c.errMu.RLock()
	defer c.errMu.RUnlock()

	return errors.Trace(c.err)
}

func (c *Client) Subscribe(ctx context.Context) <-chan Message {
	msgChan := make(chan Message)

	go func() {
		for {
			message, err := c.read(ctx)
			if err != nil {
				c.errMu.Lock()
				close(msgChan)
				c.err = errors.Trace(err)
				c.errMu.Unlock()
				return
			}

			msgChan <- message
		}
	}()

	return msgChan
}
