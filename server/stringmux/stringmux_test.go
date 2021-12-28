package stringmux_test

import (
	"io"
	"net"
	"testing"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/stringmux"
	"github.com/peer-calls/peer-calls/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

var addr1 = &net.UDPAddr{
	IP:   net.IPv4(127, 0, 0, 1),
	Port: 1234,
}

var addr2 = &net.UDPAddr{
	IP:   net.IPv4(127, 0, 0, 1),
	Port: 5678,
}

func TestStringMuxNew(t *testing.T) {
	goleak.VerifyNone(t)
	defer goleak.VerifyNone(t)

	conn1, err := net.DialUDP("udp", addr1, addr2)
	require.NoError(t, err)

	conn2, err := net.DialUDP("udp", addr2, addr1)
	require.NoError(t, err)

	sm1 := stringmux.New(stringmux.Params{
		Conn:         conn1,
		Log:          test.NewLogger(),
		ReadChanSize: 8,
	})
	defer sm1.Close()

	sm2 := stringmux.New(stringmux.Params{
		Conn:         conn2,
		Log:          test.NewLogger(),
		ReadChanSize: 8,
	})
	defer sm2.Close()

	c1, err := sm1.GetConn("ab")
	require.NoError(t, err)
	defer c1.Close()

	_, err = sm1.GetConn("ab")
	assert.Equal(t, errors.Cause(err), stringmux.ErrConnAlreadyExists)
	assert.Equal(t, "ab", c1.StreamID())

	i, err := c1.Write([]byte("ping"))
	require.NoError(t, err)
	assert.Equal(t, 4, i)

	c2, err := sm2.AcceptConn()
	require.NoError(t, err)
	assert.Equal(t, "ab", c2.StreamID())

	defer c2.Close()

	b := make([]byte, 10)
	i, err = c2.Read(b)
	require.NoError(t, err)
	assert.Equal(t, 4, i)
	assert.Equal(t, []byte("ping"), b[:i])

	c2.CloseWrite()
	i, err = c2.Write([]byte{1, 2, 3})
	assert.Equal(t, 0, i, "should not write any bytes")
	assert.Equal(t, io.ErrClosedPipe, errors.Cause(err), "ErrClosedPipe because write closed")
}

func TestStringMux_Close_GetConn(t *testing.T) {
	goleak.VerifyNone(t)
	defer goleak.VerifyNone(t)

	conn1, err := net.DialUDP("udp", addr1, addr2)
	require.NoError(t, err)

	sm1 := stringmux.New(stringmux.Params{
		Conn: conn1,
		Log:  test.NewLogger(),
	})

	sm1.Close()

	createdConn, err := sm1.GetConn("test")
	require.Equal(t, errors.Cause(err), io.ErrClosedPipe)
	require.Nil(t, createdConn)
}
