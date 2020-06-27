package server

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

type ControlEvent struct {
	StreamEvent *StreamEvent
	TrackEvent  *TrackEvent
	Type        ControlEventType
}

type StreamEvent struct {
	StreamID uint16
	Type     StreamEventType
}

type ControlEventType int

const (
	ControlEventTypeStream ControlEventType = iota + 1
	ControlEventTypeTrack
)

type StreamEventType int

const (
	StreamEventTypeAsk StreamEventType = iota + 1
	StreamEventTypeAllow
	StreamStreamTypeDeny
)

// ControlTransport is used for transporting metadata and handshake events for
// SCTP stream ID/room associations.
type ControlTransport struct {
	params    *ControlTransportParams
	mu        sync.Mutex
	wg        sync.WaitGroup
	closedCh  chan struct{}
	closeOnce sync.Once

	subscriberChans map[string]chan TrackEvent
	subscribers     map[string]*ServerMetadataAdapter

	streamEventsCh chan StreamEvent
}

type ControlTransportParams struct {
	Conn   io.ReadWriteCloser
	Logger Logger
}

func NewControlTransport(params ControlTransportParams) *ControlTransport {
	ct := &ControlTransport{
		params:          &params,
		subscribers:     map[string]*ServerMetadataAdapter{},
		subscriberChans: map[string]chan TrackEvent{},
		streamEventsCh:  make(chan StreamEvent),
	}

	ct.wg.Add(1)
	go func() {
		defer ct.wg.Done()
		ct.start()
	}()

	return ct
}

func (c *ControlTransport) CloseChannel() <-chan struct{} {
	return c.closedCh
}

func (c *ControlTransport) Close() error {
	c.closeOnce.Do(func() {
		c.params.Conn.Close()

		c.wg.Wait()

		c.mu.Lock()
		defer c.mu.Unlock()

		close(c.closedCh)

		for _, subscription := range c.subscribers {
			subscription.Close()
		}
	})

	return nil
}

func (c *ControlTransport) start() {
	buf := make([]byte, receiveMTU)
	for {
		i, err := c.params.Conn.Read(buf)
		if err != nil {
			c.params.Logger.Printf("Error reading from control conn: %s", err)
			return
		}

		var controlEvent ControlEvent
		err = json.Unmarshal(buf[:i], &controlEvent)

		if err != nil {
			c.params.Logger.Printf("Error unmarshaling received control event: %s", err)
			return
		}

		if controlEvent.Type == ControlEventTypeStream {
			c.streamEventsCh <- *controlEvent.StreamEvent
			continue
		}

		if controlEvent.TrackEvent != nil {
			roomIdentifiable, ok := controlEvent.TrackEvent.TrackInfo.Track.(RoomIdentifiable)
			if !ok {
				c.params.Logger.Printf("Got track event but don't know which room it belongs to")
				continue
			}

			room := roomIdentifiable.RoomID()
			if subChan, ok := c.subscriberChans[room]; ok {
				subChan <- *controlEvent.TrackEvent
			}
		}
	}
}

func (c *ControlTransport) sendEvent(controlEvent ControlEvent) error {
	buf, err := json.Marshal(controlEvent)
	if err != nil {
		return fmt.Errorf("Error marshalling controlEvent: %w", err)
	}

	_, err = c.params.Conn.Write(buf)
	return err
}

func (c *ControlTransport) StreamEventsChannel() <-chan StreamEvent {
	return c.streamEventsCh
}

func (c *ControlTransport) SendStreamEvent(streamEvent StreamEvent) error {
	return c.sendEvent(ControlEvent{
		StreamEvent: &streamEvent,
		Type:        ControlEventTypeStream,
	})
}

func (c *ControlTransport) sendTrackEvent(trackEvent TrackEvent) error {
	return c.sendEvent(ControlEvent{
		TrackEvent: &trackEvent,
		Type:       ControlEventTypeTrack,
	})
}

// Subscribe subscribes to track events in a specific room. There can only
// be one subscriber per room.
func (c *ControlTransport) Subscribe(room string) (io.ReadWriteCloser, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.closedCh:
		return nil, fmt.Errorf("ControlTransport is closed")
	default:
	}

	if _, ok := c.subscribers[room]; ok {
		return nil, fmt.Errorf("Already subscribed to room: %s", room)
	}

	subscriberChan := make(chan TrackEvent)
	var closeOnce sync.Once

	adapter := NewServerMetadataAdapter(ServerMetadataAdapterParams{
		Write:    c.sendTrackEvent,
		ReadChan: subscriberChan,
		Close: func() error {
			closeOnce.Do(func() {
				close(subscriberChan)

				c.mu.Lock()
				defer c.mu.Unlock()

				delete(c.subscribers, room)
				delete(c.subscriberChans, room)
			})
			return nil
		},
	})

	c.subscribers[room] = adapter
	c.subscriberChans[room] = subscriberChan

	return adapter, nil
}

// ServerMetadataAdapter is an adapter between ControlTransport and
// ServerMetadataTransport.
type ServerMetadataAdapter struct {
	params *ServerMetadataAdapterParams
}

type ServerMetadataAdapterParams struct {
	Write    func(TrackEvent) error
	Close    func() error
	ReadChan <-chan TrackEvent
}

func NewServerMetadataAdapter(params ServerMetadataAdapterParams) *ServerMetadataAdapter {
	return &ServerMetadataAdapter{
		params: &params,
	}
}

var _ io.ReadWriteCloser = &ServerMetadataAdapter{}

func (t *ServerMetadataAdapter) Write(buf []byte) (int, error) {
	var trackEvent TrackEvent

	err := json.Unmarshal(buf, &trackEvent)
	if err != nil {
		return 0, fmt.Errorf("Error unmarshalling trackEvent: %w", err)
	}

	err = t.params.Write(trackEvent)
	if err != nil {
		return 0, fmt.Errorf("Error sending trackEvent: %w", err)
	}

	return len(buf), nil
}

func (t *ServerMetadataAdapter) Read(buf []byte) (int, error) {
	trackEvent, ok := <-t.params.ReadChan

	if !ok {
		return 0, fmt.Errorf("Read channel closed")
	}

	src, err := json.Marshal(trackEvent)

	if err != nil {
		return 0, fmt.Errorf("Error marshalling read trackEvent: %w", err)
	}

	copy(buf, src)
	return len(src), nil
}

func (t *ServerMetadataAdapter) Close() error {
	return t.params.Close()
}
