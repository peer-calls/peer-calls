package udptransport

import (
	"context"

	"github.com/juju/errors"
)

type Request struct {
	cancel func()

	context      context.Context
	streamID     string
	responseChan chan Response
	torndown     chan struct{}
	setChan      chan Response
}

type Response struct {
	Transport *Transport
	Err       error
}

func NewRequest(ctx context.Context, streamID string) *Request {
	ctx, cancel := context.WithCancel(ctx)

	t := &Request{
		context:      ctx,
		cancel:       cancel,
		streamID:     streamID,
		responseChan: make(chan Response, 1),
		torndown:     make(chan struct{}),
		setChan:      make(chan Response),
	}

	go t.start(ctx)

	return t
}

func (t *Request) Context() context.Context {
	return t.context
}

func (t *Request) Cancel() {
	t.cancel()
}

func (t *Request) StreamID() string {
	return t.streamID
}

func (t *Request) start(ctx context.Context) {
	defer close(t.torndown)

	select {
	case <-ctx.Done():
		t.responseChan <- Response{
			Err:       errors.Trace(ctx.Err()),
			Transport: nil,
		}
	case res := <-t.setChan:
		t.responseChan <- res
	}
}

func (t *Request) set(streamTransport *Transport, err error) {
	res := Response{
		Transport: streamTransport,
		Err:       err,
	}

	select {
	case t.setChan <- res:
	case <-t.torndown:
	}
}

func (t *Request) Response() <-chan Response {
	return t.responseChan
}

func (t *Request) Done() <-chan struct{} {
	return t.torndown
}
