package servertransport

import (
	"fmt"
	"io"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/pion/webrtc/v3"
)

type DataTransport struct {
	params DataTransportParams

	messagesChan chan webrtc.DataChannelMessage
}

var _ transport.DataTransport = &DataTransport{}

type DataTransportParams struct {
	Log  logger.Logger
	Conn io.ReadWriteCloser
}

func NewDataTransport(params DataTransportParams) *DataTransport {
	params.Log = params.Log.WithNamespaceAppended("server_data_transport")

	transport := &DataTransport{
		params:       params,
		messagesChan: make(chan webrtc.DataChannelMessage),
	}

	go transport.start()

	return transport
}

func (t *DataTransport) start() {
	defer close(t.messagesChan)

	buf := make([]byte, ReceiveMTU)

	for {
		i, err := t.params.Conn.Read(buf)
		if err != nil {
			t.params.Log.Error("Read remote data", errors.Trace(err), nil)

			return
		}

		if i < 1 {
			t.params.Log.Error(fmt.Sprintf("Message too short: %d", i), nil, nil)

			return
		}

		// This is a little wasteful as a whole byte is being used as a boolean,
		// but works for now.
		isString := !(buf[0] == 0)

		// TODO figure out which user a message belongs to.
		message := webrtc.DataChannelMessage{
			IsString: isString,
			Data:     buf[1:],
		}

		t.messagesChan <- message
	}
}

func (t *DataTransport) MessagesChannel() <-chan webrtc.DataChannelMessage {
	return t.messagesChan
}

func (t *DataTransport) Send(message webrtc.DataChannelMessage) <-chan error {
	b := make([]byte, 0, len(message.Data)+1)

	if message.IsString {
		// Mark as string
		b = append(b, 1)
	} else {
		// Mark as binary
		b = append(b, 0)
	}

	b = append(b, message.Data...)

	_, err := t.params.Conn.Write(b)

	errCh := make(chan error, 1)
	errCh <- err

	return errCh
}

func (t *DataTransport) SendText(message string) error {
	b := make([]byte, 0, len(message)+1)
	// mark as string
	b = append(b, 1)
	b = append(b, message...)

	_, err := t.params.Conn.Write(b)

	return errors.Trace(err)
}

func (t *DataTransport) Close() error {
	return t.params.Conn.Close()
}
