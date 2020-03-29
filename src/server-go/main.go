package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/jeremija/peer-calls/src/server-go/config"
	"github.com/jeremija/peer-calls/src/server-go/factory/adapter"
	"github.com/jeremija/peer-calls/src/server-go/iceauth"
	"github.com/jeremija/peer-calls/src/server-go/logger"
	"github.com/jeremija/peer-calls/src/server-go/room"
	"github.com/jeremija/peer-calls/src/server-go/routes"
	"github.com/jeremija/peer-calls/src/server-go/server"
	"github.com/jeremija/peer-calls/src/server-go/wrtc/tracks"
)

var gitDescribe string = "v0.0.0"

func panicOnError(err error, message string) {
	if err != nil {
		panic(fmt.Errorf("%s: %w", message, err))
	}
}

var log = logger.GetLogger("main")

func main() {
	flags := flag.NewFlagSet("start", flag.ExitOnError)
	var configFilename string
	flags.StringVar(&configFilename, "c", "", "Config file to use")
	flags.Parse(os.Args)

	var c config.Config
	if configFilename != "" {
		err := config.ReadFiles([]string{configFilename}, &c)
		panicOnError(err, "Error reading config file")
	}
	if len(c.ICEServers) == 0 {
		c.ICEServers = append(c.ICEServers, config.ICEServer{
			URLs:     []string{"stun:stun.l.google.com:19302"},
			AuthType: config.AuthTypeNone,
		})
	}
	config.ReadEnv("PEERCALLS_", &c)

	ice, err := json.Marshal(iceauth.GetICEServers(c.ICEServers))
	log.Printf("Using ice servers: %s", ice)
	panicOnError(err, "Error setting ICE servers")
	newAdapter := adapter.NewAdapterFactory(c.Store)
	rooms := room.NewRoomManager(newAdapter.NewAdapter)
	tracks := tracks.NewTracksManager()
	mux := routes.NewMux(c.BaseURL, gitDescribe, c.ICEServers, string(ice), rooms, tracks)
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
