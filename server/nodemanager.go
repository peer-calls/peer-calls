package server

import (
	"net"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
)

type NodeManager struct {
	params           *NodeManagerParams
	wg               sync.WaitGroup
	mu               sync.Mutex
	transportManager *TransportManager
	rooms            map[string]struct{}
}

type NodeManagerParams struct {
	Log           logger.Logger
	RoomManager   *ChannelRoomManager
	TracksManager TracksManager
	ListenAddr    *net.UDPAddr
	Nodes         []*net.UDPAddr
}

func NewNodeManager(params NodeManagerParams) (*NodeManager, error) {
	params.Log = params.Log.WithNamespaceAppended("node_manager").WithCtx(logger.Ctx{
		"local_addr": params.ListenAddr,
	})

	conn, err := net.ListenUDP("udp", params.ListenAddr)
	if err != nil {
		return nil, errors.Annotatef(err, "listen udp: %s", params.ListenAddr)
	}

	params.Log.Info("Listen on UDP", nil)

	transportManager := NewTransportManager(TransportManagerParams{
		Conn: conn,
		Log:  params.Log,
	})

	nm := &NodeManager{
		params:           &params,
		transportManager: transportManager,
		rooms:            map[string]struct{}{},
	}

	for _, addr := range params.Nodes {
		log := params.Log.WithCtx(logger.Ctx{
			"remote_addr": addr,
		})

		log.Info("Configuring remote node", nil)

		factory, err := transportManager.GetTransportFactory(addr)
		if err != nil {
			log.Error("Create transport factory", errors.Trace(err), nil)
		}

		nm.handleTransportFactory(factory)
	}

	go nm.startTransportEventLoop()
	go nm.startRoomEventLoop()

	return nm, nil
}

func (nm *NodeManager) startTransportEventLoop() {
	for {
		factory, err := nm.transportManager.AcceptTransportFactory()
		if err != nil {
			nm.params.Log.Error("Accept transport factory", errors.Trace(err), nil)

			return
		}

		nm.handleTransportFactory(factory)
	}
}

func (nm *NodeManager) handleTransportFactory(factory *TransportFactory) {
	nm.wg.Add(1)

	go func() {
		defer nm.wg.Done()

		doneChan := make(chan struct{})
		closeChannelOnce := sync.Once{}

		done := func() {
			closeChannelOnce.Do(func() {
				close(doneChan)
			})
		}

		for {
			select {
			case <-doneChan:
				nm.params.Log.Info("Aborting server transport factory goroutine", nil)

				return
			default:
			}

			req := factory.AcceptTransport()
			errChan := nm.handleTransportRequest(req)

			nm.wg.Add(1)

			go func() {
				defer nm.wg.Done()

				err := <-errChan
				if err != nil {
					nm.params.Log.Error("Wait for transport request", errors.Trace(err), nil)
					done()
				}
			}()
		}
	}()
}

func (nm *NodeManager) handleTransportRequest(req *TransportRequest) <-chan error {
	errChan := make(chan error, 1)

	nm.wg.Add(1)

	go func() {
		defer nm.wg.Done()
		defer close(errChan)

		response := <-req.Response()

		if err := response.Err; err != nil {
			errChan <- err
			nm.params.Log.Error("Transport promise", errors.Trace(err), nil)
			return
		}

		streamTransport := response.Transport

		nm.mu.Lock()
		defer nm.mu.Unlock()

		nm.params.Log.Info("Add transport", logger.Ctx{
			"stream_id": req.StreamID(),
			"client_id": streamTransport.ClientID(),
		})
		nm.params.TracksManager.Add(req.StreamID(), streamTransport)
	}()

	return errChan
}

func (nm *NodeManager) startRoomEventLoop() {
	for {
		roomEvent, err := nm.params.RoomManager.AcceptEvent()
		if err != nil {
			nm.params.Log.Error("Accept room event", errors.Trace(err), nil)

			return
		}

		switch roomEvent.Type {
		case RoomEventTypeAdd:
			for _, factory := range nm.transportManager.Factories() {
				nm.params.Log.Info("Creating new transport", logger.Ctx{
					"room": roomEvent.RoomName,
				})
				transportRequest := factory.NewTransport(roomEvent.RoomName)
				nm.handleTransportRequest(transportRequest)
			}
		case RoomEventTypeRemove:
			for _, factory := range nm.transportManager.Factories() {
				nm.params.Log.Info("Closing transport", logger.Ctx{
					"room": roomEvent.RoomName,
				})
				factory.CloseTransport(roomEvent.RoomName)
			}
		}
	}
}

func (nm *NodeManager) Close() error {
	nm.params.RoomManager.Close()
	nm.transportManager.Close()

	nm.wg.Wait()

	return nil
}
