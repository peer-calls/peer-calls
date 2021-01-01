package udptransport

import (
	"io"
	"net"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/servertransport"
	"github.com/peer-calls/peer-calls/server/stringmux"
	"github.com/peer-calls/peer-calls/server/udpmux"
)

// Manager is in charge of managing server-to-server UDP Transports. The
// overarching design is as follows.
//
//  1. UDPMux is used to demultiplex UDP packets coming in from different Peer
//     Calls nodes based on remote addr.
//  2. For each incoming server packet from a specific remote address, a new
//     transport factory is created. A transport factory can also be created
//     manually.
//  3. Each factory creates a separate transport peer room, and it uses the
//     stringmux package to figure out which packets are for which room.
//  4. Each stream transport then uses a stringmux again to figure out which
//     packet is for which transport component:
//     - packets with 'm' prefix are media packets for MediaTransport, and
//     - packets with 's' prefix are for SCTP component which is used for
//       DataTransport and MetadataTransport.
//
// Due to the issues with sctp connection closure, it might be wise to create
// long-lived SCTP connection per factory and demultiplex packets separately.
type Manager struct {
	params *ManagerParams

	// udpMux is used for demultiplexing UDP packets from other server nodes.
	udpMux *udpmux.UDPMux

	// torndown will be closed when manager is closed.
	torndown chan struct{}

	// factoriesChan contains accepted Factories.
	factoriesChan chan *Factory
	closeOnce     sync.Once
	mu            sync.RWMutex
	wg            sync.WaitGroup

	// factories is the map of all created and active Factories.
	factories map[*stringmux.StringMux]*Factory
}

// ManagerParams are the parameters for Manager.
type ManagerParams struct {
	// Conn is the packet connection to use for sending server-to-server data.
	Conn net.PacketConn
	Log  logger.Logger
}

// NewManager creates a new instance of Manager.
func NewManager(params ManagerParams) *Manager {
	params.Log = params.Log.WithNamespaceAppended("transport_manager")

	readChanSize := 100

	udpMux := udpmux.New(udpmux.Params{
		Conn:           params.Conn,
		MTU:            uint32(servertransport.ReceiveMTU),
		Log:            params.Log,
		ReadChanSize:   readChanSize,
		ReadBufferSize: 0,
	})

	t := &Manager{
		params:        &params,
		udpMux:        udpMux,
		torndown:      make(chan struct{}),
		factoriesChan: make(chan *Factory),
		factories:     make(map[*stringmux.StringMux]*Factory),
	}

	t.wg.Add(1)

	go func() {
		defer t.wg.Done()
		t.start()
	}()

	return t
}

func (t *Manager) Factories() []*Factory {
	t.mu.RLock()
	defer t.mu.RUnlock()

	factories := make([]*Factory, 0, len(t.factories))

	for _, factory := range t.factories {
		factories = append(factories, factory)
	}

	return factories
}

func (t *Manager) start() {
	for {
		conn, err := t.udpMux.AcceptConn()
		if err != nil {
			t.params.Log.Error("Accept UDPMux conn", errors.Trace(err), nil)

			return
		}

		log := t.params.Log.WithCtx(logger.Ctx{
			"remote_addr": conn.RemoteAddr(),
		})

		log.Info("Accept UDP conn", nil)

		factory, err := t.createFactory(conn)
		if err != nil {
			t.params.Log.Error("Create Transport Factory", errors.Trace(err), nil)

			return
		}

		t.factoriesChan <- factory
	}
}

// createFactory creates a new Factory for the provided
// connection.
func (t *Manager) createFactory(conn net.Conn) (*Factory, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	readChanSize := 100

	stringMux := stringmux.New(stringmux.Params{
		Log:            t.params.Log,
		Conn:           conn,
		MTU:            uint32(servertransport.ReceiveMTU), // TODO not sure if this is ok
		ReadChanSize:   readChanSize,
		ReadBufferSize: 0,
	})

	factory := NewFactory(t.params.Log, &t.wg, stringMux)
	t.factories[stringMux] = factory

	t.wg.Add(1)

	go func() {
		defer t.wg.Done()
		<-stringMux.Done()

		t.mu.Lock()
		defer t.mu.Unlock()

		delete(t.factories, stringMux)
	}()

	return factory, nil
}

func (t *Manager) AcceptFactory() (*Factory, error) {
	factory, ok := <-t.factoriesChan
	if !ok {
		return nil, errors.Annotate(io.ErrClosedPipe, "Manager is tearing down")
	}

	return factory, nil
}

func (t *Manager) GetFactory(raddr net.Addr) (*Factory, error) {
	conn, err := t.udpMux.GetConn(raddr)
	if err != nil {
		return nil, errors.Annotatef(err, "getting conn for raddr: %s", raddr)
	}

	return t.createFactory(conn)
}

func (t *Manager) Close() error {
	err := t.close()

	t.wg.Wait()

	return err
}

func (t *Manager) Done() <-chan struct{} {
	return t.torndown
}

func (t *Manager) close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	err := t.udpMux.Close()

	t.closeOnce.Do(func() {
		close(t.factoriesChan)

		for stringMux, factory := range t.factories {
			_ = stringMux.Close()

			factory.Close()

			delete(t.factories, stringMux)
		}

		close(t.torndown)
	})

	return errors.Trace(err)
}
