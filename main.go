package main

import (
	pkgErrors "errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server"
	"github.com/peer-calls/peer-calls/server/logformatter"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/sfu"
)

const gitDescribe string = "v0.0.0"

func configure(log logger.Logger, args []string) (net.Listener, *server.StartStopper, error) {
	flags := flag.NewFlagSet("peer-calls", flag.ExitOnError)

	var configFilename string

	flags.StringVar(&configFilename, "c", "", "Config file to use")

	if err := flags.Parse(args); err != nil {
		return nil, nil, errors.Annotate(err, "parse flags")
	}

	configFiles := []string{}
	if configFilename != "" {
		configFiles = append(configFiles, configFilename)
	}
	c, err := server.ReadConfig(configFiles)
	if err != nil {
		return nil, nil, errors.Annotate(err, "read config")
	}

	log.Info(fmt.Sprintf("Using config: %+v", c), nil)
	// rooms := server.NewAdapterRoomManager(newAdapter.NewAdapter)
	tracks := sfu.NewTracksManager(log, c.Network.SFU.JitterBuffer)

	roomManagerFactory := server.NewRoomManagerFactory(server.RoomManagerFactoryParams{
		AdapterFactory: server.NewAdapterFactory(log, c.Store),
		Log:            log,
		TracksManager:  tracks,
	})
	rooms, _ := roomManagerFactory.NewRoomManager(c.Network)

	mux := server.NewMux(log, c.BaseURL, gitDescribe, c.Network, c.ICEServers, rooms, tracks, c.Prometheus)
	l, err := net.Listen("tcp", net.JoinHostPort(c.BindHost, strconv.Itoa(c.BindPort)))
	if err != nil {
		return nil, nil, errors.Annotate(err, "listen")
	}
	startStopper := server.NewStartStopper(server.ServerParams{
		TLSCertFile: c.TLS.Cert,
		TLSKeyFile:  c.TLS.Key,
	}, mux)
	return l, startStopper, nil
}

func start(args []string) (addr *net.TCPAddr, stop func() error, errChan <-chan error) {
	log := logger.New().
		WithConfig(
			logger.NewConfig(logger.ConfigMap{
				"**:sdp":          logger.LevelError,
				"**:ws":           logger.LevelError,
				"**:nack":         logger.LevelError,
				"**:signaller:**": logger.LevelError,
				"**:pion:**":      logger.LevelWarn,
				"**:pubsub":       logger.LevelTrace,
				"**:factory":      logger.LevelTrace,
				"":                logger.LevelInfo,
			}),
		).
		WithConfig(logger.NewConfigFromString(os.Getenv("PEERCALLS_LOG"))). // FIXME use wildcards here
		WithFormatter(logformatter.New()).
		WithNamespaceAppended("main")

	ch := make(chan error, 1)
	l, startStopper, err := configure(log, args)
	if err != nil {
		ch <- errors.Annotate(err, "configure")
		close(ch)
		return nil, nil, ch
	}

	addr = l.Addr().(*net.TCPAddr)
	log.Info("Listen", logger.Ctx{
		"local_addr": addr,
	})

	go func() {
		err := startStopper.Start(l)
		if !pkgErrors.Is(errors.Cause(err), http.ErrServerClosed) {
			ch <- errors.Annotate(err, "start server")
		}

		close(ch)
	}()

	return addr, startStopper.Stop, ch
}

func main() {
	_, _, errChan := start(os.Args[1:])
	err := <-errChan
	if err != nil {
		fmt.Println("Error starting server: %w", err)
		os.Exit(1)
	}
}
