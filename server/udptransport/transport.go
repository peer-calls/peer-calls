package udptransport

import (
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/multierr"
	"github.com/peer-calls/peer-calls/server/servertransport"
	"github.com/peer-calls/peer-calls/server/stringmux"
	"github.com/pion/sctp"
)

type Transport struct {
	*servertransport.Transport

	StreamID string

	association *sctp.Association
	stringMux   *stringmux.StringMux
}

func (t *Transport) Close() error {
	errs := multierr.New()

	errs.Add(errors.Annotate(t.Transport.Close(), "close transport"))
	errs.Add(errors.Annotate(t.association.Close(), "close association"))
	errs.Add(errors.Annotate(t.stringMux.Close(), "close string mux"))

	return errors.Annotate(errs.Err(), "close stream transport")
}
