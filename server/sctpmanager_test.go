package server_test

import (
	"context"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/peer-calls/peer-calls/server"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/pion/sctp"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func listenUDP(laddr *net.UDPAddr) *net.UDPConn {
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		panic(err)
	}
	return conn
}

func createSCTPManager() *server.SCTPManager {
	loggerFactory := logger.NewFactoryFromEnv("PEERCALLS_", os.Stdout)
	udpConn := listenUDP(&net.UDPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 0,
	})

	return server.NewSCTPManager(server.SCTPManagerParams{loggerFactory, udpConn})
}

func TestSCTPManager_Close(t *testing.T) {
	goleak.VerifyNone(t)
	defer goleak.VerifyNone(t)

	m := createSCTPManager()
	defer m.Close()
}

func TestSCTPManager_AcceptAssociation_Close(t *testing.T) {
	goleak.VerifyNone(t)
	defer goleak.VerifyNone(t)

	m := createSCTPManager()

	var err error
	var association *server.Association
	ch := make(chan struct{})
	go func() {
		association, err = m.AcceptAssociation()
		close(ch)
	}()

	m.Close()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	assert.Nil(t, association)
	assert.EqualError(t, err, "SCTPManager closed")
}

func TestSCTPManager_AcceptAssociation(t *testing.T) {
	goleak.VerifyNone(t)
	defer goleak.VerifyNone(t)

	m1 := createSCTPManager()
	defer m1.Close()

	m2 := createSCTPManager()
	defer m2.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	ctx, cancel := context.WithCancel(context.Background())

	var err1, err2 error
	var a1, a2 *server.Association

	go func() {
		a1, err1 = m1.AcceptAssociation()
		assert.NoError(t, err1)
		assert.NotNil(t, a1)
		wg.Done()
	}()

	go func() {
		a2, err2 = m2.GetAssociation(m1.LocalAddr())
		assert.NoError(t, err2)
		assert.NotNil(t, a2)
		wg.Done()
	}()

	go func() {
		wg.Wait()
		cancel()
	}()

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestSCTPManager_AcceptAssociation_TwoStreams(t *testing.T) {
	goleak.VerifyNone(t)
	defer goleak.VerifyNone(t)

	m1 := createSCTPManager()
	defer m1.Close()

	m2 := createSCTPManager()
	defer m2.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	ctx, cancel := context.WithCancel(context.Background())

	var err1, err2 error
	var a1, a2 *server.Association

	go func() {
		a1, err1 = m1.AcceptAssociation()
		assert.NoError(t, err1)
		assert.NotNil(t, a1)
		wg.Done()
	}()

	go func() {
		a2, err2 = m2.GetAssociation(m1.LocalAddr())
		assert.NoError(t, err2)
		assert.NotNil(t, a2)
		wg.Done()
	}()

	go func() {
		wg.Wait()
		cancel()
	}()

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	ping1 := "ping1"
	pong1 := "pong1"
	ping2 := "ping2"
	pong2 := "oing2"

	a1s1 := sctpOpen(t, a1, 1)
	sctpSend(t, a1s1, ping1)

	a2s2 := sctpOpen(t, a2, 2)
	sctpSend(t, a2s2, ping2)

	a2s1 := sctpAccept(t, a2, 1)
	sctpRecv(t, a2s1, ping1)

	a1s2 := sctpAccept(t, a1, 2)
	sctpRecv(t, a1s2, ping2)

	sctpSend(t, a1s2, pong2)
	sctpRecv(t, a2s2, pong2)

	sctpSend(t, a2s1, pong1)
	sctpRecv(t, a1s1, pong1)
}

func sctpOpen(t *testing.T, association *server.Association, streamID uint16) *sctp.Stream {
	stream, err := association.OpenStream(streamID, sctp.PayloadTypeWebRTCBinary)
	assert.NoError(t, err)
	return stream
}

func sctpAccept(t *testing.T, association *server.Association, expectedStreamID uint16) *sctp.Stream {
	t.Helper()
	stream, err := association.AcceptStream()
	assert.NoError(t, err)
	assert.Equal(t, expectedStreamID, stream.StreamIdentifier())

	return stream
}

func sctpSend(t *testing.T, stream *sctp.Stream, value string) {
	t.Helper()
	i, err := stream.WriteSCTP([]byte(value), sctp.PayloadTypeWebRTCBinary)
	assert.NoError(t, err)
	assert.Equal(t, len(value), i)
}

func sctpRecv(t *testing.T, stream *sctp.Stream, expected string) {
	t.Helper()
	buf := make([]byte, 100)
	i, p, err := stream.ReadSCTP(buf)
	assert.NoError(t, err)
	assert.Equal(t, sctp.PayloadTypeWebRTCBinary, p)
	assert.Equal(t, len(expected), i)
	assert.Equal(t, expected, string(buf[:i]))
}
