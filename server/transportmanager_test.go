package server_test

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/peer-calls/peer-calls/server"
	"github.com/peer-calls/peer-calls/server/test"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v2/pkg/media"
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

func TestTransportManager_RTP(t *testing.T) {
	goleak.VerifyNone(t)
	defer goleak.VerifyNone(t)

	loggerFactory := test.NewLoggerFactory()

	udpConn1 := listenUDP(&net.UDPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 0,
	})
	defer udpConn1.Close()

	udpConn2 := listenUDP(&net.UDPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 0,
	})
	defer udpConn2.Close()

	var f1, f2 *server.ServerTransportFactory

	tm1 := server.NewTransportManager(server.TransportManagerParams{
		Conn:          udpConn1,
		LoggerFactory: loggerFactory,
	})
	defer tm1.Close()

	tm2 := server.NewTransportManager(server.TransportManagerParams{
		Conn:          udpConn2,
		LoggerFactory: loggerFactory,
	})
	defer tm2.Close()

	sample := media.Sample{Data: []byte{0x00, 0x01, 0x02}, Samples: 1}

	var vp8Packetizer = rtp.NewPacketizer(
		1200,
		96,
		12345678,
		&codecs.VP8Payloader{},
		rtp.NewRandomSequencer(),
		96000,
	)

	rtpPackets := vp8Packetizer.Packetize(sample.Data, sample.Samples)
	require.Equal(t, 1, len(rtpPackets), "expected only a single RTP packet")

	// Make the equality assertions pass below since this is nil.
	rtpPackets[0].CSRC = make([]uint32, 0)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		var err error

		f1, err = tm1.AcceptTransportFactory()
		require.NoError(t, err)

		transport, err := f1.AcceptTransport().Wait()
		require.NoError(t, err)
		assert.Equal(t, "test-stream", transport.StreamID)

		for _, rtpPacket := range rtpPackets {
			i, err := transport.WriteRTP(rtpPacket)
			assert.NoError(t, err)
			assert.Equal(t, rtpPacket.MarshalSize(), i, "expected to send RTP bytes")
		}

		defer transport.Close()
	}()

	go func() {
		defer wg.Done()
		var err error
		f2, err = tm2.GetTransportFactory(udpConn1.LocalAddr())
		require.NoError(t, err)

		transport, err := f2.NewTransport("test-stream").WaitTimeout(2 * time.Second)
		require.NoError(t, err)

		select {
		case pkt := <-transport.RTPChannel():
			assert.Equal(t, rtpPackets[0], pkt)
		case <-time.After(time.Second):
			assert.Fail(t, "Timed out waiting for rtp.Packet")
		}

		defer transport.Close()
	}()

	wg.Wait()
}

func TestTransportManager_NewTransport_Cancel(t *testing.T) {
	goleak.VerifyNone(t)
	defer goleak.VerifyNone(t)

	loggerFactory := test.NewLoggerFactory()

	udpConn1 := listenUDP(&net.UDPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 0,
	})
	defer udpConn1.Close()

	tm1 := server.NewTransportManager(server.TransportManagerParams{
		Conn:          udpConn1,
		LoggerFactory: loggerFactory,
	})
	defer tm1.Close()

	var err error
	f2, err := tm1.GetTransportFactory(udpConn1.LocalAddr())
	require.NoError(t, err)

	transportPromise := f2.NewTransport("test-stream")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		transport, err := transportPromise.WaitTimeout(2 * time.Second)
		require.Equal(t, server.ErrCanceled, err)
		require.Nil(t, transport)
	}()

	transportPromise.Cancel()

	wg.Wait()
}
