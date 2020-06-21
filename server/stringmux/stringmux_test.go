package stringmux_test

import (
	"net"
	"testing"

	"github.com/peer-calls/peer-calls/server/stringmux"
	"github.com/peer-calls/peer-calls/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

var loggerFactory = test.NewLoggerFactory()

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
		Conn:          conn1,
		LoggerFactory: loggerFactory,
	})
	defer sm1.Close()

	sm2 := stringmux.New(stringmux.Params{
		Conn:          conn2,
		LoggerFactory: loggerFactory,
	})
	defer sm2.Close()

	c1, err := sm1.GetConn("ab")
	require.NoError(t, err)
	defer c1.Close()

	_, err = sm1.GetConn("ab")
	assert.EqualError(t, err, "Connection already exists")
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
}
