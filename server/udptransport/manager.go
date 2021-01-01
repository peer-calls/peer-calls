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

// Manager is in charge of managing server-to-server transports.
type Manager struct {
	params        *ManagerParams
	udpMux        *udpmux.UDPMux
	closeChan     chan struct{}
	factoriesChan chan *Factory
	closeOnce     sync.Once
	mu            sync.Mutex
	wg            sync.WaitGroup

	factories map[*stringmux.StringMux]*Factory
}

type ManagerParams struct {
	Conn net.PacketConn
	Log  logger.Logger
}

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
		closeChan:     make(chan struct{}),
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
	t.mu.Lock()
	defer t.mu.Unlock()

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
func (t *Manager) createFactory(conn udpmux.Conn) (*Factory, error) {
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
		<-stringMux.CloseChannel()

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

func (t *Manager) CloseChannel() <-chan struct{} {
	return t.closeChan
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

		close(t.closeChan)
	})

	return errors.Trace(err)
}
