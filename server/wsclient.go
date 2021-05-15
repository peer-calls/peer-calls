package server

import (
	"context"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/identifiers"
	"github.com/peer-calls/peer-calls/server/message"
	"github.com/peer-calls/peer-calls/server/uuid"
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
	id         identifiers.ClientID
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

func NewClientWithID(conn WSReadWriter, id identifiers.ClientID) *Client {
	if id == "" {
		id = identifiers.ClientID(uuid.New())
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
func (c *Client) WriteTimeout(ctx context.Context, timeout time.Duration, msg message.Message) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	data, err := c.serializer.Serialize(msg)
	if err != nil {
		return errors.Annotate(err, "serialize")
	}

	err = c.conn.Write(ctx, websocket.MessageText, data)
	return errors.Annotate(err, "write")
}

func (c *Client) ID() identifiers.ClientID {
	return c.id
}

// Write writes a message to client socket
func (c *Client) Write(msg message.Message) error {
	err := c.WriteTimeout(context.Background(), defaultWSTimeout, msg)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (c *Client) read(ctx context.Context) (msg message.Message, err error) {
	typ, data, err := c.conn.Read(ctx)
	if err != nil {
		return msg, errors.Annotate(err, "read")
	}

	msg, err = c.serializer.Deserialize(data)
	if err != nil {
		return msg, errors.Annotate(err, "deserialize")
	}

	if typ != websocket.MessageText {
		return msg, errors.Errorf("unexpected text message type, but got %s", typ)
	}

	return msg, nil
}

func (c *Client) Err() error {
	c.errMu.RLock()
	defer c.errMu.RUnlock()

	return errors.Trace(c.err)
}

func (c *Client) Subscribe(ctx context.Context) <-chan message.Message {
	msgChan := make(chan message.Message)

	go func() {
		var (
			err error
			msg message.Message
		)

	loop:
		for {
			msg, err = c.read(ctx)
			if err != nil {
				err = errors.Trace(err)

				break loop
			}

			select {
			case msgChan <- msg:
			case <-ctx.Done():
				err = errors.Trace(ctx.Err())

				break loop
			}
		}

		c.errMu.Lock()
		close(msgChan)
		c.err = errors.Trace(err)
		c.errMu.Unlock()
	}()

	return msgChan
}
