package udptransport2

import (
	"io"
	"net"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/servertransport"
	"github.com/peer-calls/peer-calls/server/stringmux"
	"github.com/peer-calls/peer-calls/server/udpmux"
)

type Manager struct {
	params *ManagerParams

	newFactoryRequests chan newFactoryRequest
	factoriesRequests  chan factoriesRequest

	factories chan *Factory

	teardown chan struct{}
	torndown chan struct{}
}

type ManagerParams struct {
	Conn net.PacketConn
	Log  logger.Logger
}

func NewManager(params ManagerParams) *Manager {
	params.Log = params.Log.WithNamespaceAppended("udptransport_manager")
	params.Log = params.Log.WithCtx(logger.Ctx{
		"local_addr": params.Conn.LocalAddr(),
	})

	m := &Manager{
		params: &params,

		newFactoryRequests: make(chan newFactoryRequest),
		factoriesRequests:  make(chan factoriesRequest),

		factories: make(chan *Factory),

		teardown: make(chan struct{}),
		torndown: make(chan struct{}),
	}

	go m.start()

	return m
}

// FactoriesChannel contains factories created from incoming connections.
// Users must read from this channel to prevent deadlocks.
func (m *Manager) FactoriesChannel() <-chan *Factory {
	return m.factories
}

func (m *Manager) start() {
	readChanSize := 100

	udpMux := udpmux.New(udpmux.Params{
		Conn:           m.params.Conn,
		MTU:            uint32(servertransport.ReceiveMTU),
		Log:            m.params.Log,
		ReadChanSize:   readChanSize,
		ReadBufferSize: 0,
	})

	factories := map[string]*Factory{}

	defer func() {
		for _, f := range factories {
			f.Close()
		}

		udpMux.Close()

		close(m.factories)
		close(m.torndown)
	}()

	createFactory := func(conn net.Conn) *Factory {
		log := m.params.Log.WithCtx(logger.Ctx{
			"remote_addr": conn.RemoteAddr(),
		})

		stringMux := stringmux.New(stringmux.Params{
			Log:            m.params.Log,
			Conn:           conn,
			MTU:            uint32(servertransport.ReceiveMTU), // TODO not sure if this is ok
			ReadChanSize:   readChanSize,
			ReadBufferSize: 0,
		})

		factory := NewFactory(FactoryParams{
			Log: log,
			Mux: stringMux,
		})

		factories[conn.RemoteAddr().String()] = factory

		return factory
	}

	handleNewFactoryRequest := func(raddr net.Addr) (*Factory, error) {
		if _, ok := factories[raddr.String()]; ok {
			return nil, errors.Errorf("factory already exists: %s", raddr)
		}

		conn, err := udpMux.GetConn(raddr)
		if err != nil {
			return nil, errors.Trace(err)
		}

		factory := createFactory(conn)

		return factory, nil
	}

	for {
		select {
		case conn, ok := <-udpMux.Conns():
			if !ok {
				m.params.Log.Warn("UDPMux closed", nil)

				return
			}

			factory := createFactory(conn)

			m.factories <- factory
		case req := <-m.newFactoryRequests:
			factory, err := handleNewFactoryRequest(req.raddr)
			req.res <- newFactoryResponse{
				factory: factory,
				err:     err,
			}
			close(req.res)
		case req := <-m.factoriesRequests:
			res := make([]*Factory, 0, len(factories))

			for _, f := range factories {
				res = append(res, f)
			}

			req.res <- res
			close(req.res)
		case <-m.teardown:
			return
		}
	}
}

func (m *Manager) GetFactory(raddr net.Addr) (*Factory, error) {
	req := newFactoryRequest{
		raddr: raddr,
		res:   make(chan newFactoryResponse, 1),
	}

	select {
	case m.newFactoryRequests <- req:
		res := <-req.res

		return res.factory, errors.Trace(res.err)
	case <-m.torndown:
		return nil, errors.Trace(io.ErrClosedPipe)
	}
}

func (m *Manager) Factories() []*Factory {
	req := factoriesRequest{
		res: make(chan []*Factory, 1),
	}

	select {
	case m.factoriesRequests <- req:
		res := <-req.res

		return res
	case <-m.torndown:
		return nil
	}
}

func (m *Manager) Close() {
	select {
	case m.teardown <- struct{}{}:
		<-m.torndown
	case <-m.torndown:
	}
}

type newFactoryRequest struct {
	raddr net.Addr
	res   chan newFactoryResponse
}

type newFactoryResponse struct {
	factory *Factory
	err     error
}

type factoriesRequest struct {
	res chan []*Factory
}
