package udptransport2

import (
	"encoding/json"
	"io"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/logger"
)

type controlTransport struct {
	stream io.ReadWriteCloser

	log logger.Logger

	readEventsCh  chan controlEvent
	writeEventsCh chan controlEvent

	readLoopDone  chan struct{}
	teardown      chan struct{}
	writeLoopDone chan struct{}
}

func newControlTransport(
	log logger.Logger,
	stream io.ReadWriteCloser,
) *controlTransport {
	c := &controlTransport{
		stream: stream,
		log:    log.WithNamespaceAppended("control"),

		readEventsCh:  make(chan controlEvent),
		writeEventsCh: make(chan controlEvent),

		readLoopDone:  make(chan struct{}),
		writeLoopDone: make(chan struct{}),
		teardown:      make(chan struct{}),
	}

	go c.startReadLoop()
	go c.startWriteLoop()

	return c
}

func (c *controlTransport) startReadLoop() {
	defer func() {
		close(c.readLoopDone)
	}()

	buf := make([]byte, 1024)

	for {
		i, err := c.stream.Read(buf)
		if err != nil {
			c.log.Error("Read", errors.Trace(err), nil)

			return
		}

		var event controlEvent

		err = json.Unmarshal(buf[:i], &event)
		if err != nil {
			c.log.Error("Unmarshal", errors.Trace(err), nil)

			continue
		}

		select {
		case c.readEventsCh <- event:
		case <-c.writeLoopDone:
			return
		}
	}
}

func (c *controlTransport) startWriteLoop() {
	defer func() {
		close(c.writeLoopDone)
	}()

	handleWrite := func(event controlEvent) bool {
		b, err := json.Marshal(event)
		if err != nil {
			c.log.Error("Marshal", errors.Trace(err), nil)

			return true
		}

		_, err = c.stream.Write(b)
		if err != nil {
			c.log.Error("Write", errors.Trace(err), nil)

			return false
		}

		return true
	}

	for {
		select {
		case event := <-c.writeEventsCh:
			if !handleWrite(event) {
				return
			}
		case <-c.teardown:
			return
		}
	}
}

func (c *controlTransport) Events() <-chan controlEvent {
	return c.readEventsCh
}

func (c *controlTransport) Send(event controlEvent) error {
	select {
	case c.writeEventsCh <- event:
		return nil
	case <-c.writeLoopDone:
		return errors.Trace(io.ErrClosedPipe)
	}
}

func (c *controlTransport) Close() error {
	err := c.stream.Close()

	select {
	case c.teardown <- struct{}{}:
		<-c.writeLoopDone
	case <-c.writeLoopDone:
	}

	<-c.readLoopDone

	return errors.Trace(err)
}

type controlEvent struct {
	RemoteControlEvent *remoteControlEvent `json:"remoteControlEvent"`
	Ping               bool                `json:"ping"`
}
