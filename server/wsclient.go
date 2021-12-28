package server

import (
	"context"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/message"
	"github.com/peer-calls/peer-calls/v4/server/uuid"
	"nhooyr.io/websocket"
)

const defaultWSTimeout = 5 * time.Second

type WSWriter interface {
	Write(ctx context.Context, typ websocket.MessageType, msg []byte) error
}

type WSReader interface {
	Read(ctx context.Context) (websocket.MessageType, []byte, error)
}

type WSCloser interface {
	Close(statusCode websocket.StatusCode, reason string) error
}

type WSReadWriter interface {
	WSReader
	WSWriter
	WSCloser
}

// An abstraction for sending out to websocket using channels.
type Client struct {
	id         identifiers.ClientID
	conn       WSReadWriter
	metadata   string
	serializer ByteSerializer

	messages  chan message.Message
	closed    chan struct{}
	closeOnce sync.Once

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

	c := &Client{
		id:       id,
		conn:     conn,
		messages: make(chan message.Message),
		closed:   make(chan struct{}),
	}

	go c.readLoop()

	return c
}

func (c *Client) SetMetadata(metadata string) {
	c.metadata = metadata
}

func (c *Client) Metadata() string {
	return c.metadata
}

// Writes a message to websocket.
func (c *Client) WriteCtx(ctx context.Context, msg message.Message) error {
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

// Write writes a message to client socket with the default timeout.
func (c *Client) Write(msg message.Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultWSTimeout)
	defer cancel()

	err := c.WriteCtx(ctx, msg)
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

// Err returns the read error that might have occurred. It should be called
// after the Messages channel is closed.
func (c *Client) Err() error {
	c.errMu.RLock()
	defer c.errMu.RUnlock()

	return errors.Trace(c.err)
}

// Messages returns the read messages.
func (c *Client) Messages() <-chan message.Message {
	return c.messages
}

func (c *Client) readLoop() {
	defer close(c.messages)

	var (
		err error
		msg message.Message
	)

	for {
		msg, err = c.read(context.Background())
		if err != nil {
			c.errMu.Lock()
			c.err = errors.Trace(err)
			c.errMu.Unlock()

			break
		}

		select {
		case c.messages <- msg:
		case <-c.closed:
			// Do not block on send after calling Close. But we must keep reading
			// for graceful closure. So only break the loop after read returns an
			// error.
		}
	}
}

// Close invokes Close on the underlying websocket connection.
func (c *Client) Close(statusCode websocket.StatusCode, reason string) error {
	var err error

	c.closeOnce.Do(func() {
		err = c.conn.Close(statusCode, reason)

		close(c.closed)

		err = errors.Trace(err)
	})

	return errors.Trace(err)
}
