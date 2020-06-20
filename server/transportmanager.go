package server

import (
	"fmt"
	"net"
	"sync"

	"github.com/pion/sctp"
)

// TransportManager is in charge of managing server-to-server transports.
type TransportManager struct {
	params      *TransportManagerParams
	sctpManager *SCTPManager
	closeOnce   sync.Once
	mu          sync.Mutex
	wg          sync.WaitGroup

	rooms        map[string][]*StreamTransport
	associations map[*Association]*ServerTransportFactory
}

type StreamTransport struct {
	Transport
	StreamID uint16
}

type ServerTransportFactory struct {
	loggerFactory LoggerFactory
	association   *Association
	streamCount   uint16
	Transports    map[*StreamTransport]struct{}
	freeBuckets   map[uint16]struct{}
	mu            sync.Mutex
	wg            *sync.WaitGroup
}

type TransportManagerParams struct {
	Conn          net.PacketConn
	LoggerFactory LoggerFactory
}

func NewTransportManager(params TransportManagerParams) *TransportManager {
	sctpManager := NewSCTPManager(SCTPManagerParams{
		LoggerFactory: params.LoggerFactory,
		Conn:          params.Conn,
	})

	t := &TransportManager{
		params:       &params,
		sctpManager:  sctpManager,
		associations: make(map[*Association]*ServerTransportFactory),
	}

	return t
}

// AcceptTransportFactory returns a tranpsort factory after a new Association
// is created. The factory can be used to create new ServerTransports as
// needed.
func (t *TransportManager) AcceptTransportFactory() (*ServerTransportFactory, error) {
	association, err := t.sctpManager.AcceptAssociation()
	if err != nil {
		return nil, fmt.Errorf("Error accepting transport factory: %w", err)
	}

	return t.createServerTransportFactory(association), nil
}

// createServerTransportFactory creates a new ServerTransportFactory for the
// provided association.
func (t *TransportManager) createServerTransportFactory(association *Association) *ServerTransportFactory {
	t.mu.Lock()
	defer t.mu.Unlock()

	factory := NewServerTransportFactory(t.params.LoggerFactory, &t.wg, association)
	t.associations[association] = factory

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		<-association.CloseChannel()

		t.mu.Lock()
		defer t.mu.Unlock()

		delete(t.associations, association)
	}()

	return factory
}

func (t *TransportManager) GetTransportFactory(raddr net.Addr) (*ServerTransportFactory, error) {
	association, err := t.sctpManager.GetAssociation(raddr)
	if err != nil {
		return nil, fmt.Errorf("Error getting association for raddr %s: %w", raddr, err)
	}

	return t.createServerTransportFactory(association), nil
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
		for association := range t.associations {
			_ = association.Close()
			delete(t.associations, association)
		}
	})

	return err
}

func NewServerTransportFactory(loggerFactory LoggerFactory, wg *sync.WaitGroup, association *Association) *ServerTransportFactory {
	return &ServerTransportFactory{
		loggerFactory: loggerFactory,
		association:   association,
		streamCount:   0,
		Transports:    make(map[*StreamTransport]struct{}),
		freeBuckets:   make(map[uint16]struct{}),
		wg:            wg,
	}
}

// FIXME TODO
//
// How to keep StreamIDs in sync from two nodes??
//
// Two problems:
//
// 1. node1 and node2 might pick the same StreamIDs. Perhaps the
// initiator could only be allowed to request streams from [0,
// maxuint16/2>, and the other one from [maxuint16/2, maxuint16].
//
// 2. What if node1 and node2 both decide to create 3 streams and a a
// transport for the same room?
//

func (t *ServerTransportFactory) AcceptTransport() (*StreamTransport, error) {
	// TODO wait for 3 consecutive streams to be created.
	return nil, fmt.Errorf("Not implemented")
}

func (t *ServerTransportFactory) NewTransport() (*StreamTransport, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	var streamID uint16
	found := false
	for streamIDForReuse := range t.freeBuckets {
		streamID = streamIDForReuse
		found = true
		delete(t.freeBuckets, streamIDForReuse)
		break
	}

	if !found {
		streamID = (t.streamCount*3 + 1)
		// TODO error out on overflow, but there will probably be bigger problems when that happens.
		t.streamCount++
	}

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

	streamTransport := &StreamTransport{transport, streamID}
	t.Transports[streamTransport] = struct{}{}

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		<-transport.CloseChannel()

		t.mu.Lock()
		defer t.mu.Unlock()

		t.freeBuckets[streamTransport.StreamID] = struct{}{}

		delete(t.Transports, streamTransport)
	}()

	return streamTransport, nil
}
