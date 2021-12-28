package udptransport2

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestControlTransport(t *testing.T) {
	defer goleak.VerifyNone(t)

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 0,
		Zone: "",
	})
	require.NoError(t, err)

	defer listener.Close()

	conn1, err := net.DialTCP("tcp", nil, &net.TCPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: listener.Addr().(*net.TCPAddr).Port,
		Zone: "",
	})
	require.NoError(t, err)

	defer conn1.Close()

	conn2, err := listener.AcceptTCP()
	require.NoError(t, err)

	defer conn2.Close()

	log := test.NewLogger()

	c1 := newControlTransport(log, conn1)
	c2 := newControlTransport(log, conn2)

	send := controlEvent{
		RemoteControlEvent: &remoteControlEvent{
			Type:     remoteControlEventTypeCreate,
			StreamID: "a",
		},
		Ping: false,
	}

	err = c1.Send(send)
	require.NoError(t, err)

	var recv controlEvent

	select {
	case recv = <-c2.Events():
	case <-time.After(time.Second):
		assert.Fail(t, "timed out waiting for message")
	}

	assert.Equal(t, send, recv)

	c1.Close()
	c2.Close()

	err = c1.Send(send)
	assert.Error(t, io.ErrClosedPipe, errors.Cause(err))
}
