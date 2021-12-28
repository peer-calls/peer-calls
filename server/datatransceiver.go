package server

import (
	"io"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/sfu"
	"github.com/pion/webrtc/v3"
)

type DataTransceiver struct {
	log logger.Logger

	clientID       identifiers.ClientID
	peerConnection *webrtc.PeerConnection

	// dataChannelChan will receive DataChannels from peer connection
	// OnDataChannel handler.
	dataChannelChan chan *webrtc.DataChannel

	// privateRecvMessagesChan is never closed and it is there to prevent panics
	// when a message is received, but the recvMessagesChan has already been
	// closed.
	privateRecvMessagesChan chan webrtc.DataChannelMessage

	// recvMessagesChan contains received messages. It will be closed on
	// teardown.
	recvMessagesChan chan webrtc.DataChannelMessage

	// sendMessagesChan contains messages to be sent. It is never closed.
	sendMessagesChan chan dataTransceiverMessageSend

	// teardownChan will initiate a teardown as soon as it receives a message.
	teardownChan chan struct{}

	// torndownChan will be closed as soon as teardown is complete.
	torndownChan chan struct{}
}

func NewDataTransceiver(
	log logger.Logger,
	clientID identifiers.ClientID,
	dataChannel *webrtc.DataChannel,
	peerConnection *webrtc.PeerConnection,
) *DataTransceiver {
	d := &DataTransceiver{
		log: log.WithNamespaceAppended("datatransceiver").WithCtx(logger.Ctx{
			"client_id": clientID,
		}),
		clientID:       clientID,
		peerConnection: peerConnection,

		dataChannelChan:         make(chan *webrtc.DataChannel),
		privateRecvMessagesChan: make(chan webrtc.DataChannelMessage),
		recvMessagesChan:        make(chan webrtc.DataChannelMessage),
		sendMessagesChan:        make(chan dataTransceiverMessageSend),
		teardownChan:            make(chan struct{}),
		torndownChan:            make(chan struct{}),
	}

	go d.start()

	if dataChannel != nil {
		d.handleDataChannel(dataChannel)
	}

	peerConnection.OnDataChannel(d.handleDataChannel)

	return d
}

func (d *DataTransceiver) handleDataChannel(dataChannel *webrtc.DataChannel) {
	if dataChannel.Label() == sfu.DataChannelName {
		d.dataChannelChan <- dataChannel

		dataChannel.OnMessage(func(message webrtc.DataChannelMessage) {
			d.log.Info("DataTransceiver.handleMessage", nil)

			select {
			case <-d.torndownChan:
				return
			default:
			}

			select {
			case d.privateRecvMessagesChan <- message:
				// Successfully sent.
			case <-d.torndownChan:
				// DataTransceiver has been torn down.
			}
		})
	}
}

func (d *DataTransceiver) start() {
	defer func() {
		close(d.recvMessagesChan)
		close(d.torndownChan)
	}()

	var dataChannel *webrtc.DataChannel

	handleSendMessage := func(message webrtc.DataChannelMessage) error {
		if dataChannel == nil {
			return errors.Errorf("data channel is nil")
		}

		if message.IsString {
			return errors.Annotate(dataChannel.SendText(string(message.Data)), "send text")
		}

		return errors.Annotate(dataChannel.Send(message.Data), "send bytes")
	}

	for {
		select {
		case dc := <-d.dataChannelChan:
			dataChannel = dc
		case msg := <-d.privateRecvMessagesChan:
			d.recvMessagesChan <- msg
		case msgFuture := <-d.sendMessagesChan:
			err := handleSendMessage(msgFuture.message)
			if err != nil {
				d.log.Error("Send error", errors.Trace(err), nil)

				msgFuture.errCh <- errors.Trace(err)
			}

			close(msgFuture.errCh)
		case <-d.teardownChan:
			return
		}
	}
}

func (d *DataTransceiver) MessagesChannel() <-chan webrtc.DataChannelMessage {
	return d.recvMessagesChan
}

func (d *DataTransceiver) Close() {
	d.log.Trace("DataTransceiver.Close", nil)

	select {
	case d.teardownChan <- struct{}{}:
	case <-d.torndownChan:
	}

	<-d.torndownChan
}

func (d *DataTransceiver) Send(message webrtc.DataChannelMessage) <-chan error {
	errCh := make(chan error, 1)

	select {
	case d.sendMessagesChan <- dataTransceiverMessageSend{
		errCh:   errCh,
		message: message,
	}:
	case <-d.torndownChan:
		errCh <- errors.Trace(io.ErrClosedPipe)
		close(errCh)
	}

	return errCh
}

type dataTransceiverMessageSend struct {
	// errCh will have error written to it if it occurrs. It will be closed once
	// the message sending has finished.
	errCh chan<- error
	// message to send.
	message webrtc.DataChannelMessage
}
