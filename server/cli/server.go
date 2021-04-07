package cli

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server"
	"github.com/peer-calls/peer-calls/server/command"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/sfu"
	"github.com/spf13/pflag"
)

type serverHandler struct {
	args struct {
		config string
	}

	log    logger.Logger
	config server.Config
	props  Props
	server *server.Server
	mux    *server.Mux
}

func (h *serverHandler) RegisterFlags(c *command.Command, flags *pflag.FlagSet) {
	flags.StringVarP(&h.args.config, "config", "c", "", "config file to use")
}

func (h *serverHandler) Handle(ctx context.Context, args []string) error {
	if err := h.configure(); err != nil {
		return errors.Trace(err)
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(
		h.config.BindHost,
		strconv.Itoa(h.config.BindPort),
	))
	if err != nil {
		return errors.Annotate(err, "listen")
	}

	h.server = server.New(server.Params{
		TLSCertFile: h.config.TLS.Cert,
		TLSKeyFile:  h.config.TLS.Key,
	}, h.mux)

	defer listener.Close()

	addr, _ := listener.Addr().(*net.TCPAddr)
	h.log.Info("Listen", logger.Ctx{
		"local_addr": addr,
	})

	err = h.server.Start(ctx, listener)

	return errors.Trace(err)
}

func newServerCmd(props Props) *command.Command {
	h := &serverHandler{
		log:   props.Log,
		props: props,
	}

	return command.New(command.Params{
		Name:         "server",
		Desc:         "Starts the peer-calls server (default)",
		FlagRegistry: h,
		Handler:      h,
		SubCommands:  nil,
	})
}

func (h *serverHandler) configure() (err error) {
	log := h.log

	configFiles := []string{}
	if h.args.config != "" {
		configFiles = append(configFiles, h.args.config)
	}
	h.config, err = server.ReadConfig(configFiles)
	if err != nil {
		return errors.Annotate(err, "read config")
	}

	c := h.config

	log.Info(fmt.Sprintf("Using config: %+v", c), nil)
	tracks := sfu.NewTracksManager(log, c.Network.SFU.JitterBuffer)

	roomManagerFactory := server.NewRoomManagerFactory(server.RoomManagerFactoryParams{
		AdapterFactory: server.NewAdapterFactory(log, c.Store),
		Log:            log,
		TracksManager:  tracks,
	})
	rooms, _ := roomManagerFactory.NewRoomManager(c.Network)

	h.mux = server.NewMux(log, c.BaseURL, h.props.Version, c.Network, c.ICEServers, rooms, tracks, c.Prometheus, h.props.Embed)

	return nil
}
