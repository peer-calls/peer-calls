package promise

import (
	"sync"

	"github.com/juju/errors"
)

type promise struct {
	err    error
	doneCh chan struct{}
	once   sync.Once
}

type Promise interface {
	Deferred
	Waitable
}

type Deferred interface {
	Resolve()
	Reject(err error)
}

type Waitable interface {
	Wait() error
	WaitChannel() <-chan struct{}
}

func New() Promise {
	return &promise{
		doneCh: make(chan struct{}),
	}
}

func (p *promise) done(err error) {
	p.once.Do(func() {
		p.err = errors.Trace(err)
		close(p.doneCh)
	})
}

func (p *promise) Resolve() {
	p.done(nil)
}

func (p *promise) Reject(err error) {
	p.done(errors.Trace(err))
}

func (p *promise) Wait() error {
	<-p.doneCh

	return errors.Trace(p.err)
}

func (p *promise) WaitChannel() <-chan struct{} {
	return p.doneCh
}
