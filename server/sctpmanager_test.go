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
