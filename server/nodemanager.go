package server

import (
	"net"
	"sync"

	"github.com/juju/errors"
)

type NodeManager struct {
	params           *NodeManagerParams
	logger           Logger
	wg               sync.WaitGroup
	mu               sync.Mutex
	transportManager *TransportManager
	rooms            map[string]struct{}
}

type NodeManagerParams struct {
	LoggerFactory LoggerFactory
	RoomManager   *ChannelRoomManager
	TracksManager TracksManager
	ListenAddr    *net.UDPAddr
	Nodes         []*net.UDPAddr
}

func NewNodeManager(params NodeManagerParams) (*NodeManager, error) {
	logger := params.LoggerFactory.GetLogger("nodemanager")

	conn, err := net.ListenUDP("udp", params.ListenAddr)
	if err != nil {
		return nil, errors.Annotatef(err, "listen udp: %s", params.ListenAddr)
	}

	logger.Printf("Listening on UDP port: %s", conn.LocalAddr().String())

	transportManager := NewTransportManager(TransportManagerParams{
		Conn:          conn,
		LoggerFactory: params.LoggerFactory,
	})

	nm := &NodeManager{
		params:           &params,
		transportManager: transportManager,
		logger:           logger,
		rooms:            map[string]struct{}{},
	}

	for _, addr := range params.Nodes {
		logger.Printf("Configuring remote node: %s", addr.String())

		factory, err := transportManager.GetTransportFactory(addr)
		if err != nil {
			nm.logger.Println("Error creating transport factory for remote addr: %s", addr)
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
			nm.logger.Printf("Error accepting transport factory: %+v", err)

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
				nm.logger.Printf("Aborting server transport factory goroutine")

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
					nm.logger.Printf("Error while waiting for TransportRequest: %+v", err)
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
			nm.logger.Printf("Error waiting for transport promise: %+v", err)
			return
		}

		streamTransport := response.Transport

		nm.mu.Lock()
		defer nm.mu.Unlock()

		nm.logger.Printf("Add transport: %s %s %s", req.StreamID(), streamTransport.StreamID, streamTransport.ClientID())
		nm.params.TracksManager.Add(req.StreamID(), streamTransport)
	}()

	return errChan
}

func (nm *NodeManager) startRoomEventLoop() {
	for {
		roomEvent, err := nm.params.RoomManager.AcceptEvent()
		if err != nil {
			nm.logger.Printf("Error accepting room event: %+v", err)

			return
		}

		switch roomEvent.Type {
		case RoomEventTypeAdd:
			for _, factory := range nm.transportManager.Factories() {
				nm.logger.Printf("Creating new transport for room: %s", roomEvent.RoomName)
				transportRequest := factory.NewTransport(roomEvent.RoomName)
				nm.handleTransportRequest(transportRequest)
			}
		case RoomEventTypeRemove:
			for _, factory := range nm.transportManager.Factories() {
				nm.logger.Printf("Closing transport for room: %s", roomEvent.RoomName)
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
