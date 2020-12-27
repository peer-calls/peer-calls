package server

import (
	"net"
	"sync"
	"testing"

	"github.com/peer-calls/peer-calls/server/test"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/sctp"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUDPServer() *net.UDPConn {
	laddr := &net.UDPAddr{
		Port: 1234,
		IP:   net.ParseIP("127.0.0.1"),
	}
	raddr := &net.UDPAddr{
		Port: 5678,
		IP:   net.ParseIP("127.0.0.1"),
	}

	conn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		panic(err)
	}
	return conn
}

func newUDPClient(raddr net.Addr) *net.UDPConn {
	// raddr := &net.UDPAddr{
	// 	Port: 1234,
	// 	IP:   net.ParseIP("127.0.0.1"),
	// }
	laddr := &net.UDPAddr{
		Port: 5678,
		IP:   net.ParseIP("127.0.0.1"),
	}

	conn, err := net.DialUDP("udp", laddr, raddr.(*net.UDPAddr))
	if err != nil {
		panic(err)
	}
	return conn
}

func TestUDP(t *testing.T) {
	conn1 := newUDPServer()
	conn2 := newUDPClient(conn1.LocalAddr())

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

func TestServerMediaTransport_RTP(t *testing.T) {
	conn1 := newUDPServer()
	conn2 := newUDPClient(conn1.LocalAddr())

	log := test.NewLogger()

	t1 := NewServerMediaTransport(log, conn1)
	t2 := NewServerMediaTransport(log, conn2)

	defer t1.Close()
	defer t2.Close()

	ssrc := uint32(123)

	packetizer := rtp.NewPacketizer(
		receiveMTU,
		96,
		ssrc,
		&codecs.VP8Payloader{},
		rtp.NewRandomSequencer(),
		96000,
	)

	writeSample := func(transport MediaTransport, s media.Sample) []*rtp.Packet {
		pkts := packetizer.Packetize(s.Data, s.Samples)

		for _, pkt := range pkts {
			_, err := transport.WriteRTP(pkt)
			assert.NoError(t, err)
		}

		return pkts
	}

	sentPkts := writeSample(t1, media.Sample{Data: []byte{0x01}, Samples: 1})
	require.Equal(t, 1, len(sentPkts))

	expected := map[uint16][]byte{}

	for _, pkt := range sentPkts {
		b, err := pkt.Marshal()
		require.NoError(t, err)
		expected[pkt.SequenceNumber] = b
	}

	actual := map[uint16][]byte{}

	for i := 0; i < len(sentPkts); i++ {
		pkt := <-t2.RTPChannel()
		b, err := pkt.Marshal()
		require.NoError(t, err)
		actual[pkt.SequenceNumber] = b
	}

	assert.Equal(t, expected, actual)
}

func TestServerMediaTransport_RTCP(t *testing.T) {
	conn1 := newUDPServer()
	conn2 := newUDPClient(conn1.LocalAddr())

	log := test.NewLogger()

	t1 := NewServerMediaTransport(log, conn1)
	t2 := NewServerMediaTransport(log, conn2)

	defer t1.Close()
	defer t2.Close()

	senderReport := rtcp.SenderReport{
		SSRC: uint32(123),
	}

	writeRTCP := func(transport MediaTransport, pkts []rtcp.Packet) {
		err := transport.WriteRTCP(pkts)
		require.NoError(t, err)
	}

	writeRTCP(t1, []rtcp.Packet{&senderReport})

	sentBytes, err := senderReport.Marshal()
	require.NoError(t, err)

	recvPkt := <-t2.RTCPChannel()

	recvBytes, err := recvPkt.Marshal()
	require.NoError(t, err)

	assert.Equal(t, sentBytes, recvBytes)
}

func TestServerMediaTransport_SCTP_ClientClient(t *testing.T) {
	conn1 := newUDPServer()
	conn2 := newUDPClient(conn1.LocalAddr())

	defer conn1.Close()
	defer conn2.Close()

	log := test.NewLogger()

	plf := NewPionLoggerFactory(log)

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
			MaxReceiveBufferSize: uint32(receiveMTU),
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
			MaxReceiveBufferSize: uint32(receiveMTU),
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
