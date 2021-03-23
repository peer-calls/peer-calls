package servertransport

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/peer-calls/peer-calls/server/multierr"
	"github.com/peer-calls/peer-calls/server/pionlogger"
	"github.com/peer-calls/peer-calls/server/test"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/sctp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var portCounter int64 = 10000 // nolint:gochecknoglobals

func newUDPPair() (*net.UDPConn, *net.UDPConn) {
	port1 := int(atomic.AddInt64(&portCounter, 1))
	port2 := int(atomic.AddInt64(&portCounter, 1))

	conn1 := newUDPServer(port1, port2)
	conn2 := newUDPClient(port2, conn1.LocalAddr())

	return conn1, conn2
}

func newUDPServer(localPort, remotePort int) *net.UDPConn {
	laddr := &net.UDPAddr{
		Port: localPort,
		IP:   net.ParseIP("127.0.0.1"),
		Zone: "",
	}

	raddr := &net.UDPAddr{
		Port: remotePort,
		IP:   net.ParseIP("127.0.0.1"),
		Zone: "",
	}

	conn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		panic(err)
	}

	return conn
}

func newUDPClient(localPort int, raddr net.Addr) *net.UDPConn {
	laddr := &net.UDPAddr{
		Port: localPort,
		IP:   net.ParseIP("127.0.0.1"),
		Zone: "",
	}

	conn, err := net.DialUDP("udp", laddr, raddr.(*net.UDPAddr))
	if err != nil {
		panic(err)
	}

	return conn
}

func TestUDP(t *testing.T) {
	conn1, conn2 := newUDPPair()

	defer conn1.Close()
	defer conn2.Close()

	i, err := conn1.Write([]byte("ping"))
	assert.NoError(t, err)
	assert.Equal(t, 4, i)

	buf := make([]byte, 4)
	i, err = conn2.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 4, i)
	assert.Equal(t, "ping", string(buf))

	i, err = conn2.Write([]byte("pong"))
	assert.NoError(t, err)
	assert.Equal(t, 4, i)

	i, err = conn1.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 4, i)
	assert.Equal(t, "pong", string(buf))
}

func createTransportPairs(t *testing.T) (transport.Transport, transport.Transport) {
	log := test.NewLogger()

	media1, media2 := newUDPPair()
	data1, data2 := newUDPPair()
	metadata1, metadata2 := newUDPPair()

	t.Cleanup(func() {
		media1.Close()
		media2.Close()

		data1.Close()
		data2.Close()

		metadata1.Close()
		metadata2.Close()
	})

	t1 := New(log, media1, data1, metadata1)
	t2 := New(log, media2, data2, metadata2)

	t.Cleanup(func() {
		t1.Close()
		t2.Close()
	})

	return t1, t2
}

// nolint: gochecknoglobals
var audioCodec = transport.Codec{
	MimeType:    "audio/opus",
	ClockRate:   48000,
	Channels:    2,
	SDPFmtpLine: "",
}

func TestTransport_AddTrack(t *testing.T) {
	cancel := test.Timeout(t, 10*time.Second)
	defer cancel()

	log := test.NewLogger()

	t1, t2 := createTransportPairs(t)

	track := transport.NewSimpleTrack("a", "b", audioCodec, "user1")

	localTrack, sender, err := t1.AddTrack(track)
	require.NoError(t, err)

	_ = sender

	var remoteTrack transport.TrackRemote

	select {
	case remoteTrack = <-t2.RemoteTracksChannel():
		assert.Equal(t, track, remoteTrack.Track(), "expected track details to be equal")
	case <-time.After(time.Second):
		require.FailNow(t, "timed out waiting for remote track")
	}

	log.Info("Got remote track, subscribing", nil)

	packetizer := rtp.NewPacketizer(
		ReceiveMTU,
		111,
		0,
		&codecs.OpusPayloader{},
		rtp.NewRandomSequencer(),
		audioCodec.ClockRate,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	sample := []byte{0x01, 0x02, 0x03}
	localPacket := packetizer.Packetize(sample, 1)[0]
	localPacket.CSRC = []uint32{} // just to keep the equality check correct.

	// We need to keep trying sending packets until one is received. This is
	// because packets won't be sent until there is at least one subscriber.
	go sendPacket(t, ctx.Done(), localTrack, localPacket)

	// We need to subscribe to track first to not waste bandwidth if nobody is
	// interested in the track.
	err = remoteTrack.(suber).Subscribe()
	require.NoError(t, err, "failed to subscribe to track")

	remotePacket, _, err := remoteTrack.ReadRTP()
	assert.NoError(t, err, "error reading rtp packet")
	require.NotNil(t, remotePacket, "remote packet was nil")

	assert.Equal(t, localPacket, remotePacket, "expected packets to be equal")

	cancel()

	log.Info("Read RTP", nil)

	err = t1.RemoveTrack(localTrack.Track().UniqueID())
	assert.NoError(t, err, "removing track")

	// Track should end here.
	for {
		_, _, err := remoteTrack.ReadRTP()
		if multierr.Is(err, io.EOF) {
			break
		}
	}
}

func sendPacket(
	t *testing.T,
	done <-chan struct{},
	localTrack transport.TrackLocal,
	packet *rtp.Packet,
) {
	for {
		select {
		case <-time.After(20 * time.Millisecond):
			err := localTrack.WriteRTP(packet)
			assert.NoError(t, err, "error writing rtp packet")
		case <-done:
			return
		}
	}
}

type suber interface {
	Subscribe() error
}

// type unsuber interface {
// 	Unsubscribe() error
// }

// func TestServerMediaTransport_RTCP(t *testing.T) {
// 	conn1 := newUDPServer()
// 	conn2 := newUDPClient(conn1.LocalAddr())

// 	log := test.NewLogger()

// 	t1 := NewMediaTransport(log, conn1)
// 	t2 := NewMediaTransport(log, conn2)

// 	defer t1.Close()
// 	defer t2.Close()

// 	senderReport := rtcp.SenderReport{
// 		SSRC: uint32(123),
// 	}

// 	writeRTCP := func(transport transport.RTCPWriter, pkts []rtcp.Packet) {
// 		err := transport.WriteRTCP(pkts)
// 		require.NoError(t, err)
// 	}

// 	writeRTCP(t1, []rtcp.Packet{&senderReport})

// 	sentBytes, err := senderReport.Marshal()
// 	require.NoError(t, err)

// 	recvPkts := <-t2.RTCPChannel()
// 	assert.Equal(t, 1, len(recvPkts))

// 	recvBytes, err := recvPkts[0].Marshal()
// 	require.NoError(t, err)

// 	assert.Equal(t, sentBytes, recvBytes)
// }

func TestServerMediaTransport_SCTP_ClientClient(t *testing.T) {
	conn1, conn2 := newUDPPair()

	defer conn1.Close()
	defer conn2.Close()

	log := test.NewLogger()

	plf := pionlogger.NewFactory(log)

	var wg sync.WaitGroup

	wg.Add(2)

	// SCTP needs to be started in separate goroutines because creating a new
	// client will block until the handshake is complete, and there will be no
	// handshake until both clients are created

	var c1 *sctp.Association
	go func() {
		var err error
		c1, err = sctp.Client(sctp.Config{
			NetConn:              conn1,
			MaxReceiveBufferSize: uint32(ReceiveMTU),
			MaxMessageSize:       0,
			LoggerFactory:        plf,
		})
		require.NoError(t, err)

		wg.Done()
	}()

	var c2 *sctp.Association
	go func() {
		var err error
		c2, err = sctp.Client(sctp.Config{
			NetConn:              conn2,
			MaxReceiveBufferSize: uint32(ReceiveMTU),
			MaxMessageSize:       0,
			LoggerFactory:        plf,
		})
		require.NoError(t, err)

		wg.Done()
	}()

	wg.Wait()

	t.Log("open stream")
	s1, err := c1.OpenStream(1, sctp.PayloadTypeWebRTCString)
	require.NoError(t, err)

	// need to call write before accepting stream
	t.Log("write")
	i, err := s1.Write([]byte("ping"))
	require.NoError(t, err)
	require.Equal(t, 4, i)

	t.Log("accept stream")
	s2, err := c2.AcceptStream()
	require.NoError(t, err)
	assert.Equal(t, uint16(1), s2.StreamIdentifier())

	t.Log("recv")
	buf := make([]byte, 4)
	i, err = s2.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 4, i)
	assert.Equal(t, "ping", string(buf))

	conn1.Close()
	i, err = s2.Write([]byte("second"))
	require.NoError(t, err)
	require.Equal(t, 6, i)

	// b := make([]byte, 128)
	// _, err = s2.Read(b)
	// require.NoError(t, err)
}
