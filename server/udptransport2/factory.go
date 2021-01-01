package udptransport2

import (
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/stringmux"
)

type Factory struct {
	params *FactoryParams

	teardown chan struct{}
	torndown chan struct{}
}

type FactoryParams struct {
	Log logger.Logger
	Mux *stringmux.StringMux
}

func NewFactory(params FactoryParams) *Factory {
	params.Log = params.Log.WithNamespaceAppended("udptransport_factory")

	f := &Factory{
		params: &params,

		teardown: make(chan struct{}),
		torndown: make(chan struct{}),
	}

	go f.start()

	return f
}

func (f *Factory) start() {
	defer close(f.torndown)

	for {
		select {
		case <-f.teardown:
			return
		}
	}
}

func (f *Factory) AcceptTransport() {
}

func (f *Factory) NewTransport(streamID string) {
}

func (f *Factory) Close() {
	select {
	case f.teardown <- struct{}{}:
		<-f.torndown
	case <-f.torndown:
	}
}
