package server_test

import (
	"net"
	"sync"
	"testing"

	"github.com/peer-calls/peer-calls/server"
	"github.com/peer-calls/peer-calls/server/test"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestTransportManager(t *testing.T) {
	goleak.VerifyNone(t)
	defer goleak.VerifyNone(t)

	loggerFactory := test.NewLoggerFactory()

	udpConn1 := listenUDP(&net.UDPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 0,
	})

	udpConn2 := listenUDP(&net.UDPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 0,
	})

	var f1, f2 *server.ServerTransportFactory

	tm1 := server.NewTransportManager(server.TransportManagerParams{
		Conn:          udpConn1,
		LoggerFactory: loggerFactory,
	})

	tm2 := server.NewTransportManager(server.TransportManagerParams{
		Conn:          udpConn2,
		LoggerFactory: loggerFactory,
	})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		var err error
		f1, err = tm1.AcceptTransportFactory()
		require.NoError(t, err)
	}()

	go func() {
		defer wg.Done()
		var err error
		f2, err = tm2.GetTransportFactory(udpConn1.LocalAddr())
		require.NoError(t, err)
	}()

	wg.Wait()

	_, _ = f1, f2

	tm1.Close()
	tm2.Close()
}
