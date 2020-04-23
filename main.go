package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/peer-calls/peer-calls/server/config"
	"github.com/peer-calls/peer-calls/server/factory/adapter"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/room"
	"github.com/peer-calls/peer-calls/server/routes"
	"github.com/peer-calls/peer-calls/server/server"
	"github.com/peer-calls/peer-calls/server/wrtc/tracks"
)

var gitDescribe string = "v0.0.0"

func panicOnError(err error, message string) {
	if err != nil {
		panic(fmt.Errorf("%s: %w", message, err))
	}
}

var log = logger.GetLogger("main")

func init() {
	logger.SetDefaultEnabled([]string{
		"-sdp",
		"-ws",
		"-pion:*:trace",
		"-pion:*:debug",
		"-pion:*:info",
		"*",
	})
}

func main() {
	flags := flag.NewFlagSet("peer-calls", flag.ExitOnError)
	var configFilename string
	flags.StringVar(&configFilename, "c", "", "Config file to use")
	flags.Parse(os.Args[1:])

	configFiles := []string{}
	if configFilename != "" {
		configFiles = append(configFiles, configFilename)
	}
	c, err := config.Read(configFiles)
	panicOnError(err, "Error reading config")

	log.Printf("Using config: %+v", c)
	newAdapter := adapter.NewAdapterFactory(c.Store)
	rooms := room.NewRoomManager(newAdapter.NewAdapter)
	tracks := tracks.NewTracksManager()
	mux := routes.NewMux(c.BaseURL, gitDescribe, c.Network, c.ICEServers, rooms, tracks)
	l, err := net.Listen("tcp", net.JoinHostPort(c.BindHost, strconv.Itoa(c.BindPort)))
	panicOnError(err, "Error starting server listener")
	addr := l.Addr().(*net.TCPAddr)
	log.Printf("Listening on: %s", addr.String())
	server := server.NewStartStopper(server.ServerParams{
		TLSCertFile: c.TLS.Cert,
		TLSKeyFile:  c.TLS.Key,
	}, mux)
	err = server.Start(l)
	panicOnError(err, "Error starting server")
}
