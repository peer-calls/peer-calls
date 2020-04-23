package server

import (
	"net"
	"net/http"
)

type ServerParams struct {
	TLSCertFile string
	TLSKeyFile  string
}

type StartStopper struct {
	server *http.Server
	params ServerParams
}

func NewStartStopper(params ServerParams, handler http.Handler) *StartStopper {
	server := &http.Server{
		Handler: handler,
	}
	return &StartStopper{
		server: server,
		params: params,
	}
}

func (s StartStopper) Start(l net.Listener) (err error) {
	if s.params.TLSCertFile != "" {
		err = s.server.ServeTLS(l, s.params.TLSCertFile, s.params.TLSKeyFile)
	} else {
		err = s.server.Serve(l)
	}
	return
}

func (s StartStopper) Stop() error {
	return s.server.Close()
}
