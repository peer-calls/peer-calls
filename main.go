package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/peer-calls/peer-calls/server"
	"github.com/peer-calls/peer-calls/server/logger"
)

var gitDescribe string = "v0.0.0"

func panicOnError(err error, message string) {
	if err != nil {
		panic(fmt.Errorf("%s: %w", message, err))
	}
}

func main() {
	loggerFactory := logger.NewFactoryFromEnv("PEERCALLS_", os.Stderr)
	loggerFactory.SetDefaultEnabled([]string{
		"-sdp",
		"-ws",
		"-pion:*:trace",
		"-pion:*:debug",
		"-pion:*:info",
		"*",
	})
	log := loggerFactory.GetLogger("main")

	flags := flag.NewFlagSet("peer-calls", flag.ExitOnError)
	var configFilename string
	flags.StringVar(&configFilename, "c", "", "Config file to use")
	flags.Parse(os.Args[1:])

	configFiles := []string{}
	if configFilename != "" {
		configFiles = append(configFiles, configFilename)
	}
	c, err := server.ReadConfig(configFiles)
	panicOnError(err, "Error reading config")

	log.Printf("Using config: %+v", c)
	newAdapter := server.NewAdapterFactory(loggerFactory, c.Store)
	rooms := server.NewAdapterRoomManager(newAdapter.NewAdapter)
	tracks := server.NewMemoryTracksManager(loggerFactory)
	mux := server.NewMux(loggerFactory, c.BaseURL, gitDescribe, c.Network, c.ICEServers, rooms, tracks)
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
