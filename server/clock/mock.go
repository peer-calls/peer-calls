package clock

import (
	"fmt"
	"sync"
	"time"
)

// Mock exists to allow easier mocking of Clock interface.
type Mock struct {
	mu      sync.RWMutex
	time    time.Time
	tickers map[*mockTicker]struct{}
}

var _ Clock = &Mock{}

// NewMock returns a mocked instance of a Clock.
func NewMock() *Mock {
	return &Mock{
		mu:      sync.RWMutex{},
		time:    time.Time{},
		tickers: map[*mockTicker]struct{}{},
	}
}

// Set adjusts the current time.
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
		for ts := ticker.getStart().Add(ticker.d); !ts.After(now) && !ticker.isStopped(); ts = ts.Add(ticker.d) {
			ticker.send(ts)
		}
	}
}

// Add adds the d to current time and sets the time.
func (m *Mock) Add(d time.Duration) time.Time {
	m.mu.Lock()
	ts := m.time.Add(d)
	m.set(ts)
	m.mu.Unlock()

	return ts
}

// Now implements the Clock interface.
func (m *Mock) Now() time.Time {
	m.mu.RLock()
	ts := m.time
	m.mu.RUnlock()

	return ts
}

// Since implements the Clock interface.
func (m *Mock) Since(ts time.Time) time.Duration {
	return m.Now().Sub(ts)
}

// NewTicker implements the Clock interface.
func (m *Mock) NewTicker(d time.Duration) Ticker {
	return &tickerWrapper{
		mockTicker: m.newTicker(d, false),
	}
}

// NewTimer implements the Clock interface.
func (m *Mock) NewTimer(d time.Duration) Timer {
	return m.newTicker(d, true)
}

// tickerWrapper just wraps a ticker into an interface that satisfies Ticker,
// because Stop method returns a boolean.
type tickerWrapper struct {
	*mockTicker
}

// Stop calls Stop on the mockTicker but does not return anything.
func (t *tickerWrapper) Stop() {
	t.mockTicker.Stop()
}

func (m *Mock) newTicker(d time.Duration, timer bool) *mockTicker {
	m.mu.Lock()
	ticker := &mockTicker{
		timer:   timer,
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
	// timer is set to false when this mock represents a Ticker and true when
	// it represents a ticker.
	timer bool
	// mock is used for getting the current time when resetting the timer.
	mock *Mock
	// mu protects d, start and stopped.
	mu      sync.Mutex
	d       time.Duration
	start   time.Time
	stopped bool
	c       chan time.Time
}

// C implements the Ticker and Timer interfaces.
func (m *mockTicker) C() <-chan time.Time {
	return m.c
}

func (m *mockTicker) getStart() time.Time {
	m.mu.Lock()
	start := m.start
	m.mu.Unlock()

	return start
}

func (m *mockTicker) send(ts time.Time) {
	m.mu.Lock()

	select {
	case m.c <- ts:
	default:
	}

	if m.timer {
		// Timers get stopped after first use, but tickers don't.
		m.stopped = true
	}

	m.start = ts

	m.mu.Unlock()
}

// Stop implements the Ticker and Timer interfaces.
func (m *mockTicker) Stop() bool {
	m.mu.Lock()
	justStopped := !m.stopped
	m.stopped = true
	m.mu.Unlock()

	return justStopped
}

func (m *mockTicker) isStopped() bool {
	m.mu.Lock()
	stopped := m.stopped
	m.mu.Unlock()

	return stopped
}

// Reset implements the Ticker and Timer interfaces.
func (m *mockTicker) Reset(d time.Duration) {
	now := m.mock.Now()

	m.mu.Lock()
	m.start = now
	m.d = d
	m.stopped = false
	m.mu.Unlock()
}
