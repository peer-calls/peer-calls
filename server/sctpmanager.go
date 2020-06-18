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

	associations     map[net.Addr]*asyncAssociation
	associationsChan chan *Association
	mu               sync.Mutex
	closedChan       chan struct{}
	closeOnce        sync.Once

	wg sync.WaitGroup
}

type SCTPManagerParams struct {
	LoggerFactory LoggerFactory
	Conn          net.PacketConn
}

type asyncAssociation struct {
	done        chan struct{}
	err         error
	association *Association
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
		associations:      map[net.Addr]*asyncAssociation{},
		pionLoggerFactory: NewPionLoggerFactory(params.LoggerFactory),
		closedChan:        make(chan struct{}),
		associationsChan:  make(chan *Association),
	}

	serverManager.wg.Add(1)
	go func() {
		defer serverManager.wg.Done()
		serverManager.start()
	}()

	return serverManager
}

func (s *SCTPManager) LocalAddr() net.Addr {
	return s.udpMux.LocalAddr()
}

func (s *SCTPManager) AcceptAssociation() (*Association, error) {
	association, ok := <-s.associationsChan
	if !ok {
		return nil, fmt.Errorf("SCTPManager closed")
	}
	return association, nil
}

func (s *SCTPManager) Close() error {
	err := s.close()

	s.wg.Wait()

	return err
}

func (s *SCTPManager) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var err error

	s.closeOnce.Do(func() {
		close(s.closedChan)
		close(s.associationsChan)
		err = s.udpMux.Close()
	})

	return err
}

func (s *SCTPManager) GetAssociation(raddr net.Addr) (*Association, error) {
	s.mu.Lock()

	aa, ok := s.associations[raddr]

	if ok {
		s.mu.Unlock()
		<-aa.done
		return aa.association, aa.err
	}

	conn, err := s.udpMux.GetConn(raddr)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}

	aa = s.createAssociation(conn)
	s.mu.Unlock()

	<-aa.done
	return aa.association, aa.err
}

func (s *SCTPManager) start() {
	for {
		conn, err := s.udpMux.AcceptConn()

		if err != nil {
			s.logger.Printf("Error accepting udpMux conn: %s", err)
			return
		}

		s.handleConn(conn)
	}
}

func (s *SCTPManager) handleConn(conn udpmux.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	aa := s.createAssociation(conn)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		<-aa.done
		if aa.err != nil {
			s.logger.Printf("Error creating association: %s: %s", conn.RemoteAddr(), aa.err)
			return
		}

		association := aa.association
		associationsChan := s.associationsChan

		select {
		case <-s.closedChan:
			associationsChan = nil
		default:
		}

		select {
		case associationsChan <- association:
			// OK
		case <-s.closedChan:
			s.logger.Printf("Got association in process of tearing down: %s", conn.RemoteAddr())
			_ = association.Close()
		}
	}()
}

// createAssociation creates a new sctp association. Since the sctp.Client
// blocks until an association is created, we return early and return the
// asyncAssociation. It has a done channel which will be closed after an
// association was created or an error occurred.
func (s *SCTPManager) createAssociation(conn udpmux.Conn) *asyncAssociation {
	aa := &asyncAssociation{make(chan struct{}), nil, nil}

	go func() {
		association, err := sctp.Client(sctp.Config{
			NetConn:              conn,
			LoggerFactory:        s.pionLoggerFactory,
			MaxReceiveBufferSize: uint32(receiveMTU),
		})

		if err == nil {
			aa.association = NewAssociation(association, conn)
		}
		aa.err = err
		close(aa.done)

		if err != nil {
			// error should be handled by the user of asyncAssociation
			return
		}

		// cleanup after connection is closed
		<-conn.CloseChannel()
		association.Close()
		s.removeAssociation(conn.RemoteAddr())
	}()

	s.associations[conn.RemoteAddr()] = aa
	return aa
}

func (s *SCTPManager) removeAssociation(raddr net.Addr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.associations, raddr)
}

type Association struct {
	*sctp.Association
	conn udpmux.Conn
}

func NewAssociation(association *sctp.Association, conn udpmux.Conn) *Association {
	return &Association{association, conn}
}

func (a *Association) CloseChannel() <-chan struct{} {
	return a.conn.CloseChannel()
}
