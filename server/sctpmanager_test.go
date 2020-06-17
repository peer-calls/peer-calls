package server_test

import (
	"net"
	"os"
	"testing"

	"github.com/peer-calls/peer-calls/server"
	"github.com/peer-calls/peer-calls/server/logger"
	"go.uber.org/goleak"
)

func listenUDP(laddr *net.UDPAddr) *net.UDPConn {
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		panic(err)
	}
	return conn
}

func TestSCTPManager_Close(t *testing.T) {
	goleak.VerifyNone(t)
	defer goleak.VerifyNone(t)

	loggerFactory := logger.NewFactoryFromEnv("PEERCALLS", os.Stdout)
	udpConn := listenUDP(&net.UDPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 0,
	})

	m := server.NewSCTPManager(server.SCTPManagerParams{loggerFactory, udpConn})
	defer m.Close()
}
