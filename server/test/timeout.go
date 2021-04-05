package test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/pprof"
	"testing"
	"time"
)

func Timeout(t *testing.T, d time.Duration) (cancel func()) {
	ctx, cancel := context.WithTimeout(context.Background(), d)

	go func() {
		<-ctx.Done()

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			if err := pprof.Lookup("goroutine").WriteTo(os.Stdout, 1); err != nil {
				fmt.Printf("failed to print goroutines: %v \n", err)
			}

			panic("timeout")
		}
	}()

	return cancel
}
