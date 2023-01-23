package server_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestPinger(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	count := int64(0)

	pinger := server.NewPinger(ctx, 5*time.Millisecond, func() {
		atomic.AddInt64(&count, 1)
	})

	for atomic.LoadInt64(&count) < 5 {
		require.NoError(t, ctx.Err())
		time.Sleep(1 * time.Millisecond)
	}

	pinger.ReceivePong()
	pinger.ReceivePong()

	cancel()

	pinger.ReceivePong()
}
