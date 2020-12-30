package server

import (
	"io"

	"github.com/juju/errors"
)

type subRequestType int

const (
	subRequestTypeSubscribe subRequestType = iota + 1
	subRequestTypeUnsubscribe
)

type trackEventSubRequest struct {
	clientID string
	typ      subRequestType
	done     chan trackEventSubResponse
}

type trackEventSubResponse struct {
	sub <-chan TrackEvent
	err error
}

type trackEventsSuber struct {
	subRequestsChan chan trackEventSubRequest
	torndown        chan struct{}
}

func newTrackEventsSuber(in <-chan TrackEvent) *trackEventsSuber {
	s := &trackEventsSuber{
		subRequestsChan: make(chan trackEventSubRequest),
		torndown:        make(chan struct{}),
	}

	go s.start(in)

	return s
}

func (s *trackEventsSuber) start(in <-chan TrackEvent) {
	subs := map[string]chan TrackEvent{}

	defer func() {
		for _, outCh := range subs {
			close(outCh)
		}

		close(s.torndown)
	}()

	for {
		select {
		case event, ok := <-in:
			if !ok {
				return
			}

			for _, out := range subs {
				// TODO timeout to prevent deadlock?
				out <- event
			}

		case req := <-s.subRequestsChan:
			if out, ok := subs[req.clientID]; ok {
				delete(subs, req.clientID)
				close(out)
			}

			if req.typ == subRequestTypeSubscribe {
				sub := make(chan TrackEvent)
				subs[req.clientID] = sub
				req.done <- trackEventSubResponse{
					sub: sub,
					err: nil,
				}
			}

			close(req.done)
		}
	}
}

func (s *trackEventsSuber) request(req trackEventSubRequest) (<-chan TrackEvent, error) {
	select {
	case s.subRequestsChan <- req:
		res := <-req.done
		return res.sub, errors.Trace(res.err)
	case <-s.torndown:
		return nil, errors.Trace(io.ErrClosedPipe)
	}
}

func (s *trackEventsSuber) Subscribe(clientID string) (<-chan TrackEvent, error) {
	req := trackEventSubRequest{
		typ:      subRequestTypeSubscribe,
		clientID: clientID,
		done:     make(chan trackEventSubResponse, 1),
	}

	sub, err := s.request(req)

	return sub, errors.Annotatef(err, "subscribe: %s", clientID)
}

func (s *trackEventsSuber) Unsubscribe(clientID string) error {
	req := trackEventSubRequest{
		typ:      subRequestTypeUnsubscribe,
		clientID: clientID,
		done:     make(chan trackEventSubResponse, 1),
	}

	_, err := s.request(req)

	return errors.Annotatef(err, "unsubscribe: %s", clientID)
}
