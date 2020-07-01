package server

import (
	"net"
	"sync"
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
		return nil, err
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

		nm.handleServerTransportFactory(factory)
	}

	go nm.startTransportEventLoop()
	go nm.startRoomEventLoop()

	return nm, nil
}

func (nm *NodeManager) startTransportEventLoop() {
	for {
		factory, err := nm.transportManager.AcceptTransportFactory()
		if err != nil {
			nm.logger.Printf("Error accepting transport factory: %s", err)
			return
		}

		nm.handleServerTransportFactory(factory)
	}
}

func (nm *NodeManager) handleServerTransportFactory(factory *ServerTransportFactory) {
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
			transportPromise := factory.AcceptTransport()
			nm.handleTransportPromise(transportPromise)

			nm.wg.Add(1)
			go func(p *TransportPromise) {
				defer nm.wg.Done()

				_, err := p.Wait()
				if err != nil {
					nm.logger.Printf("Error while waiting for TransportPromise: %s", err)
					done()
				}
			}(transportPromise)
		}
	}()
}

func (nm *NodeManager) handleTransportPromise(transportPromise *TransportPromise) {
	nm.wg.Add(1)

	go func() {
		defer nm.wg.Done()

		streamTransport, err := transportPromise.Wait()

		if err != nil {
			nm.logger.Printf("Error waiting for transport promise: %s", err)
			return
		}

		nm.mu.Lock()
		defer nm.mu.Unlock()

		nm.params.TracksManager.Add(transportPromise.StreamID(), streamTransport)
	}()
}

func (nm *NodeManager) startRoomEventLoop() {
	for {
		roomEvent, err := nm.params.RoomManager.AcceptEvent()
		if err != nil {
			nm.logger.Printf("Error accepting room event: %s", err)
			return
		}

		switch roomEvent.Type {
		case RoomEventTypeAdd:
			for _, factory := range nm.transportManager.Factories() {
				transportPromise := factory.NewTransport(roomEvent.RoomName)
				nm.handleTransportPromise(transportPromise)
			}
		case RoomEventTypeRemove:
			for _, factory := range nm.transportManager.Factories() {
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
