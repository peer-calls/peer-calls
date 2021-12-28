package pubsub

import (
	"io"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
)

type subRequestType int

const (
	subRequestTypeSubscribe subRequestType = iota + 1
	subRequestTypeUnsubscribe
)

type trackEventSubRequest struct {
	clientID identifiers.ClientID
	typ      subRequestType
	done     chan trackEventSubResponse
}

type trackEventSubResponse struct {
	sub <-chan PubTrackEvent
	err error
}

type events struct {
	subRequestsChan chan trackEventSubRequest
	torndown        chan struct{}
	bufferSize      int
}

func newEvents(in <-chan PubTrackEvent, bufferSize int) *events {
	s := &events{
		subRequestsChan: make(chan trackEventSubRequest),
		torndown:        make(chan struct{}),
		bufferSize:      bufferSize,
	}

	go s.start(in)

	return s
}

func (s *events) start(in <-chan PubTrackEvent) {
	subs := map[identifiers.ClientID]chan PubTrackEvent{}

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
			// Unsubscribe existing subscription.
			if out, ok := subs[req.clientID]; ok {
				delete(subs, req.clientID)
				close(out)
			}

			// Subscribe if necessary.
			if req.typ == subRequestTypeSubscribe {
				sub := make(chan PubTrackEvent, s.bufferSize)
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

func (s *events) request(req trackEventSubRequest) (<-chan PubTrackEvent, error) {
	select {
	case s.subRequestsChan <- req:
		res := <-req.done

		return res.sub, errors.Trace(res.err)
	case <-s.torndown:
		return nil, errors.Trace(io.ErrClosedPipe)
	}
}

func (s *events) Subscribe(clientID identifiers.ClientID) (<-chan PubTrackEvent, error) {
	req := trackEventSubRequest{
		typ:      subRequestTypeSubscribe,
		clientID: clientID,
		done:     make(chan trackEventSubResponse, 1),
	}

	sub, err := s.request(req)

	return sub, errors.Annotatef(err, "subscribe: %s", clientID)
}

func (s *events) Unsubscribe(clientID identifiers.ClientID) error {
	req := trackEventSubRequest{
		typ:      subRequestTypeUnsubscribe,
		clientID: clientID,
		done:     make(chan trackEventSubResponse, 1),
	}

	_, err := s.request(req)

	return errors.Annotatef(err, "unsubscribe: %s", clientID)
}

func (s *events) Done() <-chan struct{} {
	return s.torndown
}
