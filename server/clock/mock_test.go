package clock_test

import (
	"testing"
	"time"

	"github.com/peer-calls/peer-calls/v4/server/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMock(t *testing.T) {
	cl := clock.NewMock()

	start := time.Date(2021, 1, 10, 0, 0, 0, 0, time.UTC)
	cl.Set(start)

	assert.Equal(t, start, cl.Now())

	t1 := cl.NewTicker(5 * time.Second)

	cl.Add(4 * time.Second)

	t2 := cl.NewTicker(3 * time.Second)

	expectTick := func(ch <-chan time.Time, want time.Time, descr string) {
		select {
		case got := <-ch:
			assert.Equal(t, want.String(), got.String(), "expectTick: %s", descr)
		case <-time.After(time.Second):
			require.Failf(t, "expectTick", "timed out: %s", descr)
		}
	}

	expectNoTick := func(ch <-chan time.Time, descr string) {
		select {
		case got := <-ch:
			require.Failf(t, "expectNoTick", "got: %s: %s", got, descr)
		default:
		}
	}

	cl.Add(2 * time.Second)
	expectTick(t1.C(), start.Add(5*time.Second), "first tick")
	expectNoTick(t2.C(), "no tick")

	cl.Add(1 * time.Second)
	expectNoTick(t1.C(), "no tick")
	expectTick(t2.C(), start.Add(7*time.Second), "first tick of t2")

	cl.Add(1 * time.Second)
	expectNoTick(t1.C(), "no tick")
	expectNoTick(t2.C(), "no tick")

	cl.Add(1 * time.Second)
	expectNoTick(t1.C(), "no tick")
	expectNoTick(t2.C(), "no tick")

	t1.Stop()

	t3 := cl.NewTimer(2 * time.Second)

	cl.Add(1 * time.Second)
	expectNoTick(t1.C(), "no tick")
	expectTick(t2.C(), start.Add(10*time.Second), "no tick")
	expectNoTick(t3.C(), "no tick")

	t1.Reset(time.Second)

	cl.Add(1 * time.Second)
	expectTick(t1.C(), start.Add(11*time.Second), "no tick")
	expectNoTick(t2.C(), "no tick")
	expectTick(t3.C(), start.Add(11*time.Second), "tick")

	cl.Add(1 * time.Second)
	expectTick(t1.C(), start.Add(12*time.Second), "no tick")
	expectNoTick(t2.C(), "tick")
	expectNoTick(t3.C(), "no tick")

	cl.Add(1 * time.Second)
	expectTick(t1.C(), start.Add(13*time.Second), "no tick")
	expectTick(t2.C(), start.Add(13*time.Second), "no tick")
	expectNoTick(t3.C(), "no tick")

	cl.Add(10 * time.Second)
	expectTick(t1.C(), start.Add(14*time.Second), "no tick")
	expectTick(t2.C(), start.Add(16*time.Second), "no tick")

	t3.Reset(2 * time.Second)

	t1.Stop()
	t2.Stop()
	t3.Stop()

	cl.Add(10 * time.Second)

	expectNoTick(t1.C(), "no tick")
	expectNoTick(t2.C(), "no tick")
	expectNoTick(t3.C(), "no tick")
}
