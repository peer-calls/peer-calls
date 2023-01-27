package clock

import "time"

// Clock is the interface for methods from time package which can be easily
// mocked.
type Clock interface {
	// NewTicker returns a new Ticker containing a channel that will send the
	// time with a period specified by the duration argument. It adjusts the
	// intervals or drops ticks to make up for slow receivers. The duration d
	// must be greater than zero; if not, NewTicker will panic. Stop the ticker
	// to release associated resources.
	NewTicker(time.Duration) Ticker
	// NewTimer creates a new Timer that will send the current time on its
	// channel after at least duration d.
	NewTimer(time.Duration) Timer
	// Now returns the current local time.
	Now() time.Time
	// Since returns Now().Sub(ts)
	Since(ts time.Time) time.Duration
}

// New returns a new instance of unmocked Clock.
func New() Clock {
	return clock{}
}

type clock struct{}

// NewTicker implements the Clock interface.
func (c clock) NewTicker(d time.Duration) Ticker {
	return &ticker{
		ticker: time.NewTicker(d),
	}
}

// NewTimer implements the Clock interface.
func (c clock) NewTimer(d time.Duration) Timer {
	return &timer{
		timer: time.NewTimer(d),
	}
}

// Now implements the Clock interface.
func (c clock) Now() time.Time {
	return time.Now()
}

// Since implements the Clock interface.
func (c clock) Since(ts time.Time) time.Duration {
	return time.Since(ts)
}

// A Ticker holds a channel that delivers `ticks' of a clock at intervals. This
// interface exists so it can be easily mocked.
type Ticker interface {
	// C returns a channel on which ticks are delivered.
	C() <-chan time.Time
	// Stop turns off a ticker. After Stop, no more ticks will be sent. Stop
	// does not close the channel, to prevent a concurrent goroutine reading from
	// the channel from seeing an erroneous "tick".
	Stop()
	// Reset stops a ticker and resets its period to the specified duration. The
	// next tick will arrive after the new period elapses.
	Reset(time.Duration)
}

type ticker struct {
	ticker *time.Ticker
}

// Stop implements the Ticker interface.
func (t *ticker) Stop() {
	t.ticker.Stop()
}

// Reset implements the Ticker interface.
func (t *ticker) Reset(d time.Duration) {
	resetIfAvailable(t.ticker, d)
}

// C implements the Ticker interface.
func (t *ticker) C() <-chan time.Time {
	return t.ticker.C
}

var _ Ticker = &ticker{}

// The Timer type represents a single event. When the Timer expires, the
// current time will be sent on C. This interface exist so it can be easily
// mocked.
type Timer interface {
	// C returns a channel on which ticks are delivered.
	C() <-chan time.Time
	// Stop prevents the Timer from firing. It returns true if the call stops
	// the timer, false if the timer has already expired or been stopped. Stop
	// does not close the channel, to prevent a read from the channel succeeding
	// incorrectly.
	//
	// To ensure the channel is empty after a call to Stop, check the return
	// value and drain the channel. For example, assuming the program has not
	// received from t.C already:
	//
	// 	if !t.Stop() {
	// 		<-t.C
	// 	}
	//
	// This cannot be done concurrent to other receives from the Timer's channel
	// or other calls to the Timer's Stop method.
	//
	// For a timer created with AfterFunc(d, f), if t.Stop returns false, then
	// the timer has already expired and the function f has been started in its
	// own goroutine; Stop does not wait for f to complete before returning. If
	// the caller needs to know whether f is completed, it must coordinate with f
	// explicitly.
	Stop() bool
	// Reset changes the timer to expire after duration d. It returns true if the
	// timer had been active, false if the timer had expired or been stopped.
	//
	// Reset should be invoked only on stopped or expired timers with drained
	// channels. If a program has already received a value from t.C, the timer is
	// known to have expired and the channel drained, so t.Reset can be used
	// directly. If a program has not yet received a value from t.C, however, the
	// timer must be stopped and—if Stop reports that the timer expired
	// before being stopped—the channel explicitly drained:
	//
	// 	if !t.Stop() {
	// 		<-t.C
	// 	}
	// 	t.Reset(d)
	//
	// This should not be done concurrent to other receives from the Timer's
	// channel.
	//
	// Note that it is not possible to use Reset's return value correctly, as
	// there is a race condition between draining the channel and the new timer
	// expiring. Reset should always be invoked on stopped or expired channels,
	// as described above. The return value exists to preserve compatibility with
	// existing programs.
	Reset(time.Duration)
}

type timer struct {
	timer *time.Timer
}

// Stop implements the Timer interface.
func (t *timer) Stop() bool {
	return t.timer.Stop()
}

// Reset implements the Timer interface.
func (t *timer) Reset(d time.Duration) {
	resetIfAvailable(t.timer, d)
}

// C implements the timer interface.
func (t *timer) C() <-chan time.Time {
	return t.timer.C
}

// resetIfAvailable calls Reset method on a *time.Ticker or *time.Timer, but
// only if it exists. It panics otherwise. This is because old go versions do
// not implement Timer and the CI setup hasn't been updated yet to use go1.15.
func resetIfAvailable(t interface{}, d time.Duration) {
	ticker, ok := (t).(interface {
		Reset(time.Duration)
	})
	if !ok {
		panic("Reset not implemented in this Go version.")
	}

	ticker.Reset(d)
}
