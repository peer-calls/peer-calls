package udptransport

import (
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/multierr"
	"github.com/peer-calls/peer-calls/server/servertransport"
	"github.com/peer-calls/peer-calls/server/stringmux"
	"github.com/pion/sctp"
)

// Transport wraps servertransport.Transport and it contains the connection
// details for sending data via SCTP.
type Transport struct {
	*servertransport.Transport

	// StreamID is the StreamID from stringmux.
	StreamID string

	// association is used for creating streams for servertransport.DataTransport
	// and servertransport.MetadataTransport.
	association *sctp.Association

	// stringMux is used for demultiplexing UDP packets and directing them to
	// servertransport.MediaTransport and the SCTP association.
	stringMux *stringmux.StringMux
}

func (t *Transport) Close() error {
	errs := multierr.New()

	errs.Add(errors.Annotate(t.Transport.Close(), "close transport"))

	// FIXME closing the association is not as trivial as calling close. There
	// are two ways of closing a SCTP association: using Abort and Shutdown,
	// neither of which are currently implemented in pion/sctp. I have a MR
	// to address the Shutdown, but it hasn't yet been merged:
	//
	// https://github.com/pion/sctp/pull/176.
	//
	// Until then, we most likely need a better mechanism for supporting
	// shutdowns.
	//
	// Core issue: NodeManager will close the server association as soon as there
	// are no WebRTC connections left in the room. Eventually this Close method
	// will be called and the other end of the connection will be left hanging
	// because the Shutdown/Abort is not implemented.
	errs.Add(errors.Annotate(t.association.Close(), "close association"))
	errs.Add(errors.Annotate(t.stringMux.Close(), "close string mux"))

	return errors.Annotate(errs.Err(), "close stream transport")
}
