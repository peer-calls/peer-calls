package udptransport2_test

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/peer-calls/peer-calls/v4/server/clock"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/test"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/peer-calls/peer-calls/v4/server/udptransport2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func listenUDP(laddr *net.UDPAddr) *net.UDPConn {
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		panic(err)
	}

	return conn
}

// func waitForResponse(req *udptransport2.Request, timeout time.Duration) (*udptransport2.Transport, error) {
// 	var (
// 		transport *udptransport2.Transport
// 		err       error
// 	)

// 	timer := time.NewTimer(20 * time.Second)
// 	defer timer.Stop()

// 	select {
// 	case res := <-req.Response():
// 		transport = res.Transport
// 		err = res.Err
// 	case <-timer.C:
// 		err = errors.Errorf("timed out waiting for transport")
// 	}

// 	return transport, errors.Trace(err)
// }

func TestManager_RTP(t *testing.T) {
	goleak.VerifyNone(t)
	defer goleak.VerifyNone(t)

	log := test.NewLogger()

	udpConn1 := listenUDP(&net.UDPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 0,
		Zone: "",
	})
	defer udpConn1.Close()

	udpConn2 := listenUDP(&net.UDPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 0,
		Zone: "",
	})
	defer udpConn2.Close()

	var f1, f2 *udptransport2.Factory

	tm1 := udptransport2.NewManager(udptransport2.ManagerParams{
		Conn:           udpConn1,
		Log:            log,
		Clock:          clock.NewMock(),
		PingTimeout:    3 * time.Second,
		DestroyTimeout: 15 * time.Second,
	})
	defer tm1.Close()

	tm2 := udptransport2.NewManager(udptransport2.ManagerParams{
		Conn:           udpConn2,
		Log:            log,
		Clock:          clock.NewMock(),
		PingTimeout:    3 * time.Second,
		DestroyTimeout: 15 * time.Second,
	})
	defer tm2.Close()

	codec := transport.Codec{
		MimeType:    "audio/opus",
		ClockRate:   48000,
		Channels:    2,
		SDPFmtpLine: "",
	}

	track := transport.NewSimpleTrack("trackID", "streamID", codec, "user1")

	var wg sync.WaitGroup

	wg.Add(2)

	var transport1, transport2 *udptransport2.Transport

	go func() {
		defer wg.Done()

		fmt.Println("waiting for factory")
		f1 = <-tm1.FactoriesChannel()

		fmt.Println("waiting for transport")

		select {
		case transport1 = <-f1.TransportsChannel():
		case <-time.After(time.Second):
			require.Fail(t, "Timed out waiting for transport1")
		}

		assert.Equal(t, identifiers.RoomID("test-stream"), transport1.StreamID())

		select {
		case trwr := <-transport1.RemoteTracksChannel():
			remoteTrack := trwr.TrackRemote
			assert.Equal(t, track, remoteTrack.Track())
		case <-time.After(time.Second):
			assert.Fail(t, "Timed out waiting for track")
		}
	}()

	go func() {
		defer wg.Done()

		var err error

		fmt.Println("calling get factory", udpConn1.LocalAddr())
		f2, err = (<-tm2.GetFactory(udpConn1.LocalAddr())).Result()
		fmt.Println("got factory")
		require.NoError(t, err)

		err = f2.CreateTransport("test-stream")
		require.NoError(t, err)

		select {
		case transport2 = <-f2.TransportsChannel():
		case <-time.After(time.Second):
			require.Fail(t, "Timed out waiting for transport2")
		}

		_, _, err = transport2.AddTrack(track)
		require.NoError(t, err, "failed to add track")
	}()

	wg.Wait()

	assert.NoError(t, transport1.Close())
	assert.NoError(t, transport2.Close())

	// f1.Close()
	// f2.Close()
}

// func TestManager_NewTransport_Cancel(t *testing.T) {
// 	goleak.VerifyNone(t)
// 	defer goleak.VerifyNone(t)

// 	log := test.NewLogger()

// 	udpConn1 := listenUDP(&net.UDPAddr{
// 		IP:   net.IP{127, 0, 0, 1},
// 		Port: 0,
// 		Zone: "",
// 	})
// 	defer udpConn1.Close()

// 	tm1 := udptransport2.NewManager(udptransport2.ManagerParams{
// 		Conn: udpConn1,
// 		Log:  log,
// 	})
// 	defer tm1.Close()

// 	var err error
// 	f2, err := tm1.GetFactory(udpConn1.LocalAddr())
// 	require.NoError(t, err)

// 	transport, err := f2.NewTransport("test-stream")
// 	require.NoError(t, err, "creating transport")

// 	var wg sync.WaitGroup

// 	wg.Add(1)

// 	go func() {
// 		defer wg.Done()

// 		transport, err := waitForResponse(req, 20*time.Second)
// 		_, _ = transport, err
// 		// Do not assert here because a test might fail if a transport is created
// 		// before Cancel is called. Rare, but happens.
// 		// require.Equal(t, udptransport2.ErrCanceled, err)
// 		// require.Nil(t, transport)
// 	}()

// 	// req.Cancel()

// 	wg.Wait()
// }
