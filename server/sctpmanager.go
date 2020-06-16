package server

import (
	"fmt"
	"net"
	"sync"

	"github.com/peer-calls/peer-calls/server/udpmux"
	"github.com/pion/logging"
	"github.com/pion/sctp"
)

type SCTPManager struct {
	params            SCTPManagerParams
	logger            Logger
	udpMux            *udpmux.UDPMux
	pionLoggerFactory logging.LoggerFactory

	associations     map[net.Addr]*sctp.Association
	associationsChan chan *sctp.Association
	mu               sync.Mutex
	closedChan       chan struct{}
	closeOnce        sync.Once
}

type SCTPManagerParams struct {
	LoggerFactory LoggerFactory
	Conn          net.PacketConn
}

func NewSCTPManager(params SCTPManagerParams) *SCTPManager {
	serverManager := &SCTPManager{
		params: params,
		logger: params.LoggerFactory.GetLogger("servermanager"),
		udpMux: udpmux.New(udpmux.Params{
			Conn:          params.Conn,
			LoggerFactory: params.LoggerFactory,
			MTU:           uint32(receiveMTU),
			ReadChanSize:  100,
		}),
		associations:      map[net.Addr]*sctp.Association{},
		pionLoggerFactory: NewPionLoggerFactory(params.LoggerFactory),
		closedChan:        make(chan struct{}),
	}

	go serverManager.start()

	return serverManager
}

func (s *SCTPManager) AcceptAssociation() (*sctp.Association, error) {
	assoc, ok := <-s.associationsChan
	if !ok {
		return nil, fmt.Errorf("SCTPManager closed")
	}
	return assoc, nil
}

func (s *SCTPManager) Close() error {
	var err error

	s.closeOnce.Do(func() {
		close(s.closedChan)
		err = s.udpMux.Close()
	})

	return err
}

func (s *SCTPManager) start() {
	for {
		conn, err := s.udpMux.AcceptConn()

		if err != nil {
			s.logger.Printf("Error accepting udpMux conn: %s", err)
			return
		}

		go s.createAssocGracefully(conn)
	}
}

func (s *SCTPManager) createAssocGracefully(conn udpmux.Conn) {
	assoc, err := s.createAssociation(conn)

	if err != nil {
		s.logger.Printf("createAssocGracefully: Error creating new sctp client for remote addr: %s: %s", conn.RemoteAddr(), err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.associationsChan

	select {
	case <-s.closedChan:
		ch = nil
	default:
	}

	select {
	case ch <- assoc:
		// OK
	case <-s.closedChan:
		s.logger.Printf("createAssocGracefully new association while closing")
		assoc.Close()
	}
}

// createAssociation creates a new sctp association. This method blocks until
// connection is established.
func (s *SCTPManager) createAssociation(conn udpmux.Conn) (*sctp.Association, error) {
	association, err := sctp.Client(sctp.Config{
		NetConn:              conn,
		LoggerFactory:        s.pionLoggerFactory,
		MaxReceiveBufferSize: uint32(receiveMTU),
	})

	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.associations[conn.RemoteAddr()] = association

	go func() {
		// wait for the connection to be closed and then clean up. it is safe to
		// block here because this func will always be run in a goroutine.
		<-conn.CloseChannel()

		s.mu.Lock()
		defer s.mu.Unlock()

		assoc, ok := s.associations[conn.RemoteAddr()]
		if ok {
			_ = assoc.Close()
			delete(s.associations, conn.RemoteAddr())
		}
	}()

	// TODO handle connection close and remove from s.associations

	return association, nil
}

func (s *SCTPManager) GetAssociation(raddr net.Addr) (*sctp.Association, error) {
	// FIXME multiple simulatneous calls might overwrite associations with same
	// raddr since mutex is locked & unlocked twice.
	s.mu.Lock()
	association, ok := s.associations[raddr]
	s.mu.Unlock()

	if ok {
		return association, nil
	}

	conn, err := s.udpMux.GetConn(raddr)
	if err != nil {
		return nil, err
	}

	return s.createAssociation(conn)
}
