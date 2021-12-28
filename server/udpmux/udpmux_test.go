package udpmux

import (
	"io"
	"net"
	"testing"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/goleak"
)

func TestUDPMux_AcceptConn(t *testing.T) {
	goleak.VerifyNone(t)
	defer goleak.VerifyNone(t)

	udpConn1, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 0,
	})
	require.NoError(t, err)
	defer udpConn1.Close()

	mux := New(Params{
		Conn:         udpConn1,
		MTU:          8192,
		Log:       test.NewLogger(),
		ReadChanSize: 20,
	})
	defer mux.Close()

	conns := make(chan net.Conn)
	go func() {
		conn, err := mux.AcceptConn()
		require.NoError(t, err)

		_, err = mux.GetConn(conn.RemoteAddr())
		assert.Equal(t, errors.Cause(err), ErrConnAlreadyExists)

		conns <- conn
	}()

	udpConn2, err := net.DialUDP("udp", nil, udpConn1.LocalAddr().(*net.UDPAddr))
	require.NoError(t, err)
	defer udpConn2.Close()

	udpConn2.Write([]byte("test"))

	acceptedConn := <-conns
	defer acceptedConn.Close()

	recv := make([]byte, DefaultMTU)
	i, err := acceptedConn.Read(recv)
	require.NoError(t, err)

	assert.Equal(t, "test", string(recv[:i]))
}

func TestUDPMux_GetConn(t *testing.T) {
	goleak.VerifyNone(t)
	defer goleak.VerifyNone(t)

	udpConn1, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 0,
	})
	require.NoError(t, err)
	defer udpConn1.Close()

	udpConn2, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 0,
	})
	defer udpConn2.Close()

	mux1 := New(Params{
		Conn:           udpConn1,
		MTU:            8192,
		Log:         test.NewLogger(),
		ReadChanSize:   20,
		ReadBufferSize: 100,
	})
	defer mux1.Close()

	mux2 := New(Params{
		Conn:           udpConn2,
		MTU:            8192,
		Log:         test.NewLogger(),
		ReadChanSize:   20,
		ReadBufferSize: 100,
	})
	defer mux2.Close()

	conns := make(chan net.Conn)
	go func() {
		conn, err := mux1.AcceptConn()
		require.NoError(t, err)
		conns <- conn
	}()

	createdConn, err := mux2.GetConn(udpConn1.LocalAddr())
	require.NoError(t, err)

	createdConn.Write([]byte("test"))

	acceptedConn := <-conns
	defer acceptedConn.Close()

	recv := make([]byte, DefaultMTU)
	i, err := acceptedConn.Read(recv)
	require.NoError(t, err)

	assert.Equal(t, "test", string(recv[:i]))
}

func TestUDPMux_Close_GetConn(t *testing.T) {
	goleak.VerifyNone(t)
	defer goleak.VerifyNone(t)

	udpConn1, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 0,
	})
	require.NoError(t, err)
	defer udpConn1.Close()

	mux1 := New(Params{
		Conn:         udpConn1,
		MTU:          8192,
		Log:       test.NewLogger(),
		ReadChanSize: 20,
	})

	mux1.Close()
	<-mux1.Done()

	createdConn, err := mux1.GetConn(&net.UDPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 1234,
	})
	require.Equal(t, errors.Cause(err), io.ErrClosedPipe)
	require.Nil(t, createdConn)
}
