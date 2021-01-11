package clock

import "time"

type Clock interface {
	NewTicker(time.Duration) Ticker
	Now() time.Time
}

func New() Clock {
	return clock{}
}

type clock struct{}

func (c clock) NewTicker(d time.Duration) Ticker {
	return &ticker{
		ticker: time.NewTicker(d),
	}
}

func (c clock) Now() time.Time {
	return time.Now()
}

type Ticker interface {
	C() <-chan time.Time
	Stop()
	Reset(time.Duration)
}

type ticker struct {
	ticker *time.Ticker
}

func (t *ticker) Stop() {
	t.ticker.Stop()
}

func (t *ticker) Reset(d time.Duration) {
	ticker, ok := (interface{})(t.ticker).(interface {
		Reset(time.Duration)
	})
	if !ok {
		panic("Ticker.Reset not implemented in this Go version.")
	}

	ticker.Reset(d)
}

func (t *ticker) C() <-chan time.Time {
	return t.ticker.C
}

var _ Ticker = &ticker{}
