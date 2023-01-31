package cli

import (
	"context"
	"fmt"
	"net"
	"os"
	"path"
	"strconv"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/peer-calls/peer-calls/v4/server/command"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/sfu"
	"github.com/spf13/pflag"
)

type serverHandler struct {
	args struct {
		config    string
		pprofAddr string
	}

	log    logger.Logger
	config server.Config
	props  Props
	server *server.Server
	mux    *server.Mux
}

func (h *serverHandler) RegisterFlags(c *command.Command, flags *pflag.FlagSet) {
	flags.StringVarP(&h.args.config, "config", "c", "", "config file to use")
	flags.StringVar(&h.args.pprofAddr, "pprof-addr", "", "when set, will enable pprof server (example: 127.0.0.1:6060)")
}

func (h *serverHandler) Handle(ctx context.Context, args []string) error {
	if err := h.configure(); err != nil {
		return errors.Trace(err)
	}

	if pprofAddr := h.args.pprofAddr; pprofAddr != "" {
		pprofListener, err := net.Listen("tcp", h.args.pprofAddr)
		if err != nil {
			return errors.Annotatef(err, "listen pprof: %q", pprofAddr)
		}

		h.log.Info(fmt.Sprintf("Listen pprof %s", pprofAddr), logger.Ctx{
			"local_addr": pprofAddr,
		})

		go server.NewPProf().Start(ctx, pprofListener)
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

	if c.FS != "" {
		h.props.Embed = server.Embed{
			Templates: os.DirFS(path.Join(c.FS, "server", "templates")),
			Static:    os.DirFS(path.Join(c.FS, "build")),
			Resources: os.DirFS(path.Join(c.FS, "res")),
		}
	}

	tracks := sfu.NewTracksManager(log, c.Network.SFU.JitterBuffer)

	roomManagerFactory := server.NewRoomManagerFactory(server.RoomManagerFactoryParams{
		AdapterFactory: server.NewAdapterFactory(log, c.Store),
		Log:            log,
		TracksManager:  tracks,
	})
	rooms, _ := roomManagerFactory.NewRoomManager(c.Network)

	encodedInsertableStreams := c.Frontend.EncodedInsertableStreams

	h.mux = server.NewMux(log, c.BaseURL, h.props.Version, c.Network, c.ICEServers, encodedInsertableStreams, rooms, tracks, c.Prometheus, h.props.Embed)

	return nil
}
