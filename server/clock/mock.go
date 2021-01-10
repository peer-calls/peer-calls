package clock

import (
	"fmt"
	"sync"
	"time"
)

type Mock struct {
	mu      sync.RWMutex
	time    time.Time
	tickers map[*mockTicker]struct{}
}

var _ Clock = &Mock{}

func NewMock() *Mock {
	return &Mock{
		mu:      sync.RWMutex{},
		time:    time.Time{},
		tickers: map[*mockTicker]struct{}{},
	}
}

func (m *Mock) Set(now time.Time) {
	m.mu.Lock()
	m.set(now)
	m.mu.Unlock()
}

func (m *Mock) set(now time.Time) {
	start := m.time
	m.time = now

	diff := now.Sub(start)

	if diff < 0 {
		panic(fmt.Sprintf("diff cannot be less than zero: %d", diff))
	}

	for ticker := range m.tickers {
		ticker.mu.Lock()

		if !ticker.stopped {
			// offset := start.Sub(ticker.start) % ticker.d
			// fmt.Println("now    ", now)
			// fmt.Println("t.start", ticker.start)
			// fmt.Println("t.d    ", ticker.d)
			// fmt.Println("offset ", offset)
			for ts := ticker.start.Add(ticker.d); !ts.After(now); ts = ts.Add(ticker.d) {
				select {
				case ticker.c <- ts:
				default:
				}

				ticker.start = ts
			}
		}

		ticker.mu.Unlock()
	}
}

func (m *Mock) Add(d time.Duration) time.Time {
	m.mu.Lock()
	ts := m.time.Add(d)
	m.set(ts)
	m.mu.Unlock()

	return ts
}

func (m *Mock) Now() time.Time {
	m.mu.RLock()
	ts := m.time
	m.mu.RUnlock()

	return ts
}

func (m *Mock) NewTicker(d time.Duration) Ticker {
	m.mu.Lock()
	ticker := &mockTicker{
		c:       make(chan time.Time, 1),
		d:       d,
		mock:    m,
		start:   m.time,
		stopped: false,
	}
	m.tickers[ticker] = struct{}{}
	m.mu.Unlock()

	return ticker
}

type mockTicker struct {
	mock    *Mock
	mu      sync.Mutex
	d       time.Duration
	start   time.Time
	stopped bool
	c       chan time.Time
}

func (m *mockTicker) C() <-chan time.Time {
	return m.c
}

func (m *mockTicker) Stop() {
	m.mu.Lock()
	m.stopped = true
	m.mu.Unlock()
}

func (m *mockTicker) Reset(d time.Duration) {
	now := m.mock.Now()

	m.mu.Lock()
	m.start = now
	m.d = d
	m.stopped = false
	m.mu.Unlock()
}
