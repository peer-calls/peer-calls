package server

import (
	"fmt"
	"net"
	"sync"

	"github.com/pion/sctp"
)

type TransportManager struct {
	params         *TransportManagerParams
	logger         Logger
	sctpManager    *SCTPManager
	transportsChan chan TransportEvent
	closeChan      chan struct{}
	closeOnce      sync.Once
	mu             sync.Mutex
	wg             sync.WaitGroup

	rooms        map[string][]*RoomTransport
	associations map[*Association]*ServerTransportFactory
}

type RoomTransport struct {
	StreamID  uint16
	Transport Transport
	// Association *Association
}

type ServerTransportFactory struct {
	loggerFactory LoggerFactory
	association   *Association
	lastStreamID  uint16
	Transports    map[*RoomTransport]struct{}
	freeBuckets   map[uint16]struct{}
	mu            sync.Mutex
	wg            *sync.WaitGroup
}

func NewServerTransportFactory(loggerFactory LoggerFactory, wg *sync.WaitGroup, association *Association) *ServerTransportFactory {
	return &ServerTransportFactory{
		loggerFactory: loggerFactory,
		association:   association,
		lastStreamID:  0,
		Transports:    make(map[*RoomTransport]struct{}),
		freeBuckets:   make(map[uint16]struct{}),
		wg:            wg,
	}
}

func (t *ServerTransportFactory) NewTransport() (*RoomTransport, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	var streamID uint16
	found := false
	for streamIDForReuse := range t.freeBuckets {
		streamID = streamIDForReuse
		found = true
		break
	}

	if !found {
		t.lastStreamID++
		// TODO error out on overflow, but there will probably be bigger problems when that happens.
		streamID = t.lastStreamID
	}

	streamID = streamID * 3

	// TODO make handling of these errors nicer
	mediaStream, err := t.association.OpenStream(streamID, sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		return nil, fmt.Errorf("Error opening media stream: %w", err)
	}

	dataStream, err := t.association.OpenStream(streamID+1, sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		_ = mediaStream.Close()
		return nil, fmt.Errorf("Error opening data stream: %w", err)
	}
	metadataStream, err := t.association.OpenStream(streamID+2, sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		_ = dataStream.Close()
		_ = mediaStream.Close()
		return nil, fmt.Errorf("Error opening metadata stream: %w", err)
	}

	transport := NewServerTransport(t.loggerFactory, mediaStream, dataStream, metadataStream)

	roomTransport := &RoomTransport{streamID, transport}
	t.Transports[roomTransport] = struct{}{}

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		<-transport.CloseChannel()

		t.mu.Lock()
		defer t.mu.Unlock()

		t.freeBuckets[roomTransport.StreamID] = struct{}{}

		delete(t.Transports, roomTransport)
	}()

	return roomTransport, nil
}

type TransportEvent struct {
	Transport Transport
	Type      TransportEventType
}

type TransportEventType int

const (
	TransportEventTypeAdd TransportEventType = iota + 1
	TransportEventTypeRemove
)

type TransportManagerParams struct {
	Conn           net.PacketConn
	LoggerFactory  LoggerFactory
	RoomEventsChan <-chan RoomEvent
}

func NewTransportManager(params TransportManagerParams) *TransportManager {
	sctpManager := NewSCTPManager(SCTPManagerParams{
		LoggerFactory: nil,
		Conn:          params.Conn,
	})

	t := &TransportManager{
		params:         &params,
		sctpManager:    sctpManager,
		transportsChan: make(chan TransportEvent),
		closeChan:      make(chan struct{}),
		rooms:          make(map[string][]*RoomTransport),
		associations:   make(map[*Association]*ServerTransportFactory),
	}

	t.wg.Add(2)
	go func() {
		defer t.wg.Done()
		t.start()
	}()

	go func() {
		defer t.wg.Done()
		t.subscribeToRoomEvents()
	}()

	return t
}

func (t *TransportManager) start() {
	for {
		association, err := t.sctpManager.AcceptAssociation()

		if err != nil {
			t.logger.Printf("Error accepting association: %s", err)
			return
		}

		t.handleAssociation(association)
	}
}

func (t *TransportManager) subscribeToRoomEvents() {
	for {
		select {
		case <-t.closeChan:
			t.logger.Printf("Unsubscribing from room events since TransportManager has closed")
			return
		case roomEvent, ok := <-t.params.RoomEventsChan:
			if !ok {
				t.logger.Printf("Room events channel closed")
				return
			}

			t.handleRoomEvent(roomEvent)
		}
	}
}

func (t *TransportManager) handleRoomEvent(roomEvent RoomEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for roomEvent := range t.params.RoomEventsChan {
		switch roomEvent.Type {
		case RoomEventTypeAdd:
			// iterate through all known associations and create a ServerTransport
			// for this room. RoomManager should be listening to the transport events.
			//
			// Room manager will have to be smarter and differentiate between server
			// transports and room transports so that a room can be deleted.
			for _, factory := range t.associations {
				t.createTransport(roomEvent.RoomName, factory)
			}
		case RoomEventTypeRemove:
			// Close all transports previously created by us for this room.
			// Removal from room (should?) be handled automatically by RoomManager.
			roomTransports, ok := t.rooms[roomEvent.RoomName]
			if !ok {
				continue
			}
			for _, roomTransport := range roomTransports {
				_ = roomTransport.Transport.Close()
			}
			delete(t.rooms, roomEvent.RoomName)
		}
	}
}

// handleAssociation creates transports for all active rooms and adds the
// transports for all rooms.
func (t *TransportManager) handleAssociation(association *Association) {
	t.mu.Lock()
	defer t.mu.Unlock()

	factory, ok := t.associations[association]
	if ok {
		return
	}

	factory = NewServerTransportFactory(t.params.LoggerFactory, &t.wg, association)
	t.associations[association] = factory

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		<-association.CloseChannel()
		delete(t.associations, association)
	}()

	// Iterate through all known rooms and create transports for these associations
	for room := range t.rooms {
		t.createTransport(room, factory)
	}
}

func (t *TransportManager) AddAssociation(raddr net.Addr) error {
	association, err := t.sctpManager.GetAssociation(raddr)
	if err != nil {
		return fmt.Errorf("Error getting association for raddr %s: %w", raddr, err)
	}

	t.handleAssociation(association)
	return nil
}

func (t *TransportManager) createTransport(room string, factory *ServerTransportFactory) {
	roomTransport, err := factory.NewTransport()
	if err != nil {
		t.logger.Printf("Error creating new server transport for room: %s", room)
		return
	}

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		<-roomTransport.Transport.CloseChannel()
		t.transportsChan <- TransportEvent{roomTransport.Transport, TransportEventTypeRemove}
	}()

	t.rooms[room] = append(t.rooms[room], roomTransport)
	t.transportsChan <- TransportEvent{roomTransport.Transport, TransportEventTypeAdd}
}

func (t *TransportManager) Close() error {
	err := t.close()

	t.wg.Wait()

	return err
}

func (t *TransportManager) close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	err := t.sctpManager.Close()

	t.closeOnce.Do(func() {
		close(t.closeChan)
		close(t.transportsChan)

		for association := range t.associations {
			_ = association.Close()
			delete(t.associations, association)
		}
	})

	return err
}
