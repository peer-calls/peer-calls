package udptransport2

import (
	"io"
	"net"
	"time"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/clock"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/servertransport"
	"github.com/peer-calls/peer-calls/v4/server/udpmux"
	"github.com/pion/interceptor"
)

type Manager struct {
	params *ManagerParams

	newFactoryRequests    chan newFactoryRequest
	listFactoriesRequests chan listFactoriesRequest

	factoriesChannel chan *Factory

	teardown chan struct{}
	torndown chan struct{}
}

type ManagerParams struct {
	Conn                net.PacketConn
	Log                 logger.Logger
	Clock               clock.Clock
	PingTimeout         time.Duration
	DestroyTimeout      time.Duration
	InterceptorRegistry *interceptor.Registry
}

func NewManager(params ManagerParams) *Manager {
	params.Log = params.Log.WithNamespaceAppended("udptransport_manager")
	params.Log = params.Log.WithCtx(logger.Ctx{
		"local_addr": params.Conn.LocalAddr(),
	})

	params.Log.Trace("NewManager", nil)

	m := &Manager{
		params: &params,

		newFactoryRequests:    make(chan newFactoryRequest),
		listFactoriesRequests: make(chan listFactoriesRequest),

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

	// pendingFactories indexes Factory by raddr string.
	pendingFactoryRequests := map[string]newFactoryRequest{}

	removeFactoriesChan := make(chan string)
	createdFactoriesChan := make(chan NewFactoryResponse)

	defer func() {
		m.params.Log.Trace("Tearing down", nil)

		for raddrStr, f := range factories {
			delete(factories, raddrStr)
			f.Close()
		}

		for _, req := range pendingFactoryRequests {
			req.res <- NewFactoryResponse{
				err:     errors.Trace(io.ErrClosedPipe),
				factory: nil,
				raddr:   req.raddr.String(),
			}
		}

		udpMux.Close()

		// m.params.Conn.Close()

		close(m.factoriesChannel)
		close(m.torndown)
	}()

	createFactoryAsync := func(conn net.Conn) {
		go func() {
			log := m.params.Log.WithCtx(logger.Ctx{
				"remote_addr": conn.RemoteAddr(),
			})

			factory, err := NewFactory(FactoryParams{
				Log:                 log,
				Conn:                conn,
				Clock:               m.params.Clock,
				PingTimeout:         m.params.PingTimeout,
				InterceptorRegistry: m.params.InterceptorRegistry,
			})

			select {
			case createdFactoriesChan <- NewFactoryResponse{
				raddr:   conn.RemoteAddr().String(),
				factory: factory,
				err:     errors.Trace(err),
			}:
			case <-m.torndown:
				if factory != nil {
					factory.Close()
				}
			}
		}()
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

	_handleNewFactoryRequest := func(req newFactoryRequest) error {
		raddrStr := req.raddr.String()

		if _, ok := factories[raddrStr]; ok {
			return errors.Errorf("factory already exists: %s", req.raddr)
		}

		if _, ok := pendingFactoryRequests[raddrStr]; ok {
			return errors.Errorf("pending factory already exists: %s", req.raddr)
		}

		conn, err := udpMux.GetConn(req.raddr)
		if err != nil {
			return errors.Trace(err)
		}

		pendingFactoryRequests[raddrStr] = req

		createFactoryAsync(conn)

		return nil
	}

	handleNewFactoryRequest := func(req newFactoryRequest) {
		if err := _handleNewFactoryRequest(req); err != nil {
			req.res <- NewFactoryResponse{
				raddr:   req.raddr.String(),
				factory: nil,
				err:     errors.Trace(err),
			}
		}
	}

	handleNewFactoryResponse := func(res NewFactoryResponse) bool {
		log := m.params.Log.WithCtx(logger.Ctx{
			"remote_addr": res.raddr,
		})

		log.Trace("Handle new factory response", nil)

		if req, ok := pendingFactoryRequests[res.raddr]; ok {
			delete(pendingFactoryRequests, res.raddr)

			if res.factory != nil {
				addFactory(res.raddr, res.factory)
			}

			req.res <- res

			return true
		}

		factoriesChannel := m.factoriesChannel

		if res.err != nil {
			// Do not send a response with error to factories channel.
			factoriesChannel = nil
		}

		for {
			select {
			case factoriesChannel <- res.factory:
				m.params.Log.Debug("Accept factory", nil)

				if res.factory != nil {
					addFactory(res.raddr, res.factory)
				}

				return true
			case req := <-m.newFactoryRequests:
				if req.raddr.String() != res.raddr {
					handleNewFactoryRequest(req)

					continue
				}

				if res.factory != nil {
					addFactory(res.raddr, res.factory)
				}

				m.params.Log.Debug("Accept (get) factory", nil)

				req.res <- NewFactoryResponse{
					raddr:   res.raddr,
					factory: res.factory,
					err:     res.err,
				}

				return true
			case <-m.teardown:
				if res.factory != nil {
					res.factory.Close()
				}

				return false
			}
		}
	}

	handleConn := func(conn net.Conn) bool {
		raddrStr := conn.RemoteAddr().String()

		log := m.params.Log.WithCtx(logger.Ctx{
			"remote_addr": raddrStr,
		})

		log.Trace("Handle conn", nil)

		createFactoryAsync(conn)

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
		case res := <-createdFactoriesChan:
			if !handleNewFactoryResponse(res) {
				return
			}
		case req := <-m.listFactoriesRequests:
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

func (m *Manager) GetFactory(raddr net.Addr) <-chan NewFactoryResponse {
	req := newFactoryRequest{
		raddr: raddr,
		res:   make(chan NewFactoryResponse, 1),
	}

	select {
	case m.newFactoryRequests <- req:
		// res := <-req.res

		// return res.factory, errors.Trace(res.err)
	case <-m.torndown:
		req.res <- NewFactoryResponse{
			raddr:   raddr.String(),
			factory: nil,
			err:     errors.Trace(io.ErrClosedPipe),
		}
	}

	return req.res
}

func (m *Manager) Factories() []*Factory {
	req := listFactoriesRequest{
		res: make(chan []*Factory, 1),
	}

	select {
	case m.listFactoriesRequests <- req:
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
	res   chan NewFactoryResponse
}

type NewFactoryResponse struct {
	raddr   string
	factory *Factory
	err     error
}

func (r NewFactoryResponse) Result() (*Factory, error) {
	return r.factory, errors.Trace(r.err)
}

type listFactoriesRequest struct {
	res chan []*Factory
}
