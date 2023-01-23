package server

import (
	"context"
	"time"
)

// Pinger is a component that sends pings to clients on a regular interval and
// receives pongs back.
type Pinger struct {
	ticker *time.Ticker
	pongCh chan struct{}
	ping   func()
}

// NewPinger creates a new instance of Pinger and starts a ticker whose
// duration is set to dur. The ping callback is called on every interval. The
// main event loop will be closed when ctx is done.
func NewPinger(ctx context.Context, dur time.Duration, ping func()) *Pinger {
	ticker := time.NewTicker(dur)
	pongCh := make(chan struct{}, 1)

	p := &Pinger{
		ticker: ticker,
		pongCh: pongCh,
		ping:   ping,
	}

	go p.run(ctx)

	return p
}

// run is the main event loop.
func (p *Pinger) run(ctx context.Context) {
	defer p.ticker.Stop()
	lastPongTime := time.Time{}

	for {
		select {
		case <-p.ticker.C:
			// TODO terminate connection when we haven't received pong in a while.
			_ = lastPongTime

			p.ping()
		case <-p.pongCh:
			lastPongTime = time.Now()
		case <-ctx.Done():
			return
		}
	}
}

// ReceivePong notifies the main event loop of a new pong response in a
// non-blocking manner.
func (p *Pinger) ReceivePong() {
	select {
	case p.pongCh <- struct{}{}:
	default: // Don't block, we already have an unprocessed pong.
	}
}
