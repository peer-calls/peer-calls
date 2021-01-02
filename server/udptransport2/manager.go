package udptransport2

import (
	"io"
	"net"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/servertransport"
	"github.com/peer-calls/peer-calls/server/udpmux"
)

type Manager struct {
	params *ManagerParams

	newFactoryRequests chan newFactoryRequest
	factoriesRequests  chan listFactoriesRequest

	factoriesChannel chan *Factory

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

	params.Log.Trace("NewManager", nil)

	m := &Manager{
		params: &params,

		newFactoryRequests: make(chan newFactoryRequest),
		factoriesRequests:  make(chan listFactoriesRequest),

		factoriesChannel: make(chan *Factory),

		teardown: make(chan struct{}),
		torndown: make(chan struct{}),
	}

	go m.start()

	return m
}

// FactoriesChannel contains factories created from incoming connections.
// Users must read from this channel to prevent deadlocks.
func (m *Manager) FactoriesChannel() <-chan *Factory {
	return m.factoriesChannel
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

	// factories indexes Factory by raddr string.
	factories := map[string]*Factory{}

	removeFactoriesChan := make(chan string)

	defer func() {
		m.params.Log.Trace("Tearing down", nil)

		for raddrStr, f := range factories {
			delete(factories, raddrStr)
			f.Close()
		}

		udpMux.Close()

		close(m.factoriesChannel)
		close(m.torndown)
	}()

	createFactory := func(conn net.Conn) (*Factory, error) {
		log := m.params.Log.WithCtx(logger.Ctx{
			"remote_addr": conn.RemoteAddr(),
		})

		// TODO do not block in creating new factories because creating an SCTP
		// Association takes time.
		factory, err := NewFactory(FactoryParams{
			Log:  log,
			Conn: conn,
		})
		if err != nil {
			return nil, errors.Trace(err)
		}

		return factory, nil
	}

	addFactory := func(raddrStr string, f *Factory) {
		factories[raddrStr] = f

		go func() {
			// Remove factory automatically after it tears down.
			select {
			case <-f.Done():
			case <-m.torndown:
				return
			}

			select {
			case removeFactoriesChan <- raddrStr:
			case <-m.torndown:
			}
		}()
	}

	_handleNewFactoryRequest := func(raddr net.Addr) (*Factory, error) {
		raddrStr := raddr.String()

		if _, ok := factories[raddrStr]; ok {
			return nil, errors.Errorf("factory already exists: %s", raddr)
		}

		conn, err := udpMux.GetConn(raddr)
		if err != nil {
			return nil, errors.Trace(err)
		}

		factory, err := createFactory(conn)
		if err != nil {
			return nil, errors.Trace(err)
		}

		addFactory(raddrStr, factory)

		return factory, nil
	}

	handleNewFactoryRequest := func(req newFactoryRequest) {
		m.params.Log.Trace("New factory request start", nil)

		factory, err := _handleNewFactoryRequest(req.raddr)
		req.res <- newFactoryResponse{
			factory: factory,
			err:     errors.Trace(err),
		}

		m.params.Log.Trace("New factory request done", nil)
	}

	acceptOrGet := func(raddrStr string, factory *Factory) bool {
		for {
			select {
			case m.factoriesChannel <- factory:
				m.params.Log.Debug("Accept factory", nil)

				addFactory(raddrStr, factory)

				return true
			case req := <-m.newFactoryRequests:
				if req.raddr.String() != raddrStr {
					handleNewFactoryRequest(req)

					continue
				}

				addFactory(raddrStr, factory)

				m.params.Log.Debug("Accept (get) factory", nil)

				req.res <- newFactoryResponse{
					factory: factory,
					err:     nil,
				}

				return true
			case <-m.teardown:
				return false
			}
		}
	}

	handleConn := func(conn net.Conn) bool {
		raddrStr := conn.RemoteAddr().String()

		log := m.params.Log.WithCtx(logger.Ctx{
			"remote_addr": raddrStr,
		})

		log.Trace("Handle conn start", nil)

		factory, err := createFactory(conn)
		if err != nil {
			log.Error("Create factory", errors.Trace(err), nil)

			return true
		}

		if !acceptOrGet(raddrStr, factory) {
			return false
		}

		log.Trace("Handle conn done", nil)

		return true
	}

	for {
		select {
		case conn, ok := <-udpMux.Conns():
			if !ok {
				m.params.Log.Warn("UDPMux closed", nil)

				return
			}

			if !handleConn(conn) {
				return
			}
		case req := <-m.newFactoryRequests:
			handleNewFactoryRequest(req)
		case req := <-m.factoriesRequests:
			res := make([]*Factory, 0, len(factories))

			for _, f := range factories {
				res = append(res, f)
			}

			req.res <- res
			close(req.res)
		case raddr := <-removeFactoriesChan:
			delete(factories, raddr)
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
	req := listFactoriesRequest{
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

type listFactoriesRequest struct {
	res chan []*Factory
}
