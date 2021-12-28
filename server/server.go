package server

import (
	"context"
	"net"
	"net/http"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/multierr"
)

type Params struct {
	TLSCertFile string
	TLSKeyFile  string
}

type Server struct {
	server *http.Server
	params Params
}

func New(params Params, handler http.Handler) *Server {
	server := &http.Server{
		Handler: handler,
	}
	return &Server{
		server: server,
		params: params,
	}
}

func (s Server) Start(ctx context.Context, l net.Listener) error {
	startErrCh := make(chan error, 1)

	go func() {
		defer close(startErrCh)

		var err error

		if s.params.TLSCertFile != "" {
			err = s.server.ServeTLS(l, s.params.TLSCertFile, s.params.TLSKeyFile)
			err = errors.Trace(err)
		} else {
			err = s.server.Serve(l)
			err = errors.Trace(err)
		}

		startErrCh <- errors.Annotate(err, "start server")
	}()

	select {
	case <-ctx.Done():
	case err := <-startErrCh:
		return errors.Trace(err)
	}

	err := errors.Trace(s.server.Close())

	if startErr := <-startErrCh; startErr != nil {
		err = errors.Trace(startErr)
	}

	if !multierr.Is(err, http.ErrServerClosed) {
		return errors.Trace(err)
	}

	return nil
}
