package server_test

import (
	"fmt"
	"net"
	"sync"
	"testing"

	"github.com/peer-calls/peer-calls/server"
	"github.com/peer-calls/peer-calls/server/test"
	"github.com/stretchr/testify/assert"
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
		defer fmt.Println("done 1")
		defer wg.Done()
		var err error

		f1, err = tm1.AcceptTransportFactory()
		require.NoError(t, err)

		transport, err := f1.AcceptTransport().Wait()
		require.NoError(t, err)
		assert.Equal(t, "test-stream", transport.StreamID)
		defer transport.Close()
	}()

	go func() {
		defer wg.Done()
		var err error
		f2, err = tm2.GetTransportFactory(udpConn1.LocalAddr())
		require.NoError(t, err)

		transport, err := f2.NewTransport("test-stream").Wait()
		require.NoError(t, err)
		defer transport.Close()
	}()

	wg.Wait()

	tm1.Close()
	tm2.Close()
}
