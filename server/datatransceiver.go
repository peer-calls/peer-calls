package server

import (
	"fmt"
	"sync"

	"github.com/pion/webrtc/v2"
)

type DataTransceiver struct {
	log Logger

	clientID       string
	peerConnection *webrtc.PeerConnection

	mu             sync.RWMutex
	dataChanOnce   sync.Once
	dataChanClosed bool
	dataChannel    *webrtc.DataChannel
	messagesChan   chan webrtc.DataChannelMessage
	closeChannel   chan struct{}
}

func NewDataTransceiver(
	loggerFactory LoggerFactory,
	clientID string,
	dataChannel *webrtc.DataChannel,
	peerConnection *webrtc.PeerConnection,
) *DataTransceiver {
	d := &DataTransceiver{
		log:            loggerFactory.GetLogger("datatransceiver"),
		clientID:       clientID,
		peerConnection: peerConnection,
		messagesChan:   make(chan webrtc.DataChannelMessage),
		closeChannel:   make(chan struct{}),
	}
	if dataChannel != nil {
		d.handleDataChannel(dataChannel)
	}
	peerConnection.OnDataChannel(d.handleDataChannel)
	return d
}

func (d *DataTransceiver) handleDataChannel(dataChannel *webrtc.DataChannel) {
	d.log.Printf("[%s] DataTransceiver.handleDataChannel: %s", d.clientID, dataChannel.Label())
	if dataChannel.Label() == DataChannelName {
		// only want a single data channel for messages and sending files
		d.mu.Lock()
		dataChannel.OnMessage(d.handleMessage)
		d.dataChannel = dataChannel
		d.mu.Unlock()
	}
}

func (d *DataTransceiver) MessagesChannel() <-chan webrtc.DataChannelMessage {
	return d.messagesChan
}

func (d *DataTransceiver) Close() {
	d.log.Printf("[%s] DataTransceiver.Close", d.clientID)
	d.dataChanOnce.Do(func() {
		close(d.closeChannel)

		d.mu.Lock()
		defer d.mu.Unlock()

		d.dataChanClosed = true
		close(d.messagesChan)
	})
}

func (d *DataTransceiver) handleMessage(msg webrtc.DataChannelMessage) {
	d.log.Printf("[%s] DataTransceiver.handleMessage", d.clientID)
	d.mu.RLock()
	defer d.mu.RUnlock()

	ch := d.messagesChan
	if d.dataChanClosed {
		ch = nil
	}

	select {
	case ch <- msg:
	case <-d.closeChannel:
	}
}

func (d *DataTransceiver) SendText(message string) (err error) {
	d.log.Printf("[%s] DataTransceiver.SendText", d.clientID)
	d.mu.RLock()
	if d.dataChannel != nil {
		err = d.dataChannel.SendText(message)
	} else {
		err = fmt.Errorf("[%s] No data channel", d.clientID)
	}
	d.mu.RUnlock()
	return
}

func (d *DataTransceiver) Send(message []byte) (err error) {
	d.log.Printf("[%s] DataTransceiver.Send", d.clientID)
	d.mu.RLock()
	if d.dataChannel != nil {
		err = d.dataChannel.Send(message)
	} else {
		err = fmt.Errorf("[%s] No data channel", d.clientID)
	}
	d.mu.RUnlock()
	return
}
