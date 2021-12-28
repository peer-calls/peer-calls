package pubsub

// import (
// 	"github.com/juju/errors"
// 	"github.com/peer-calls/peer-calls/v4/server/multierr"
// 	"github.com/peer-calls/peer-calls/v4/server/transport"
// )

// type transportsMap map[string]Transport

// type pub struct {
// 	clientID string
// 	track    transport.Track
// 	subs     transportsMap
// }

// func newPub(clientID string, track transport.Track) *pub {
// 	return &pub{
// 		clientID: clientID,
// 		track:    track,
// 		subs:     transportsMap{},
// 	}
// }

// func (p *pub) sub(clientID string, transport Transport) error {
// 	if err := transport.AddTrack(p.track); err != nil {
// 		return errors.Trace(err)
// 	}

// 	p.subs[clientID] = transport

// 	var err error

// 	if t, ok := p.track.(suber); ok {
// 		// Send a message to server transport to actually subscribe to this track.
// 		// This prevents unnecessary data to be sent across different peer calls
// 		// server nodes.
// 		err = t.Sub()
// 	}

// 	return errors.Trace(err)
// }

// func (p *pub) unsub(subClientID string) error {
// 	transport, ok := p.subs[subClientID]
// 	if !ok {
// 		return errors.Trace(ErrSubNotFound)
// 	}

// 	delete(p.subs, subClientID)

// 	errs := multierr.New()

// 	if t, ok := p.track.(unsuber); ok {
// 		// Send a message to server transport to unsubscribe to this track. This
// 		// prevents unnecessary data to be sent across different peer calls server
// 		// nodes.
// 		errs.Add(t.Unsub())
// 	}

// 	errs.Add(transport.RemoveTrack(p.track.TrackID()))

// 	return errors.Trace(errs.Err())
// }

// func (p *pub) subscribers() transportsMap {
// 	return p.subs
// }

// type unsuber interface {
// 	Unsub() error
// }

// type suber interface {
// 	Sub() error
// }
