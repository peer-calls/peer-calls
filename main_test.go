package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/peer-calls/peer-calls/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartMissingConfig(t *testing.T) {
	prefix := "PEERCALLS_"
	defer test.UnsetEnvPrefix(prefix)
	os.Setenv(prefix+"BIND_PORT", "0")
	os.Setenv(prefix+"LOG", "-*")
	log := test.NewLogger()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := start(ctx, log, []string{"-c", "/missing/file.yml"})
	require.Error(t, err)
	fmt.Printf("error %+v", err)
	assert.Contains(t, err.Error(), "read config")
}

func TestStartWrongPort(t *testing.T) {
	prefix := "PEERCALLS_"
	defer test.UnsetEnvPrefix(prefix)
	os.Setenv(prefix+"BIND_PORT", "100000")
	os.Setenv(prefix+"LOG", "-*")
	log := test.NewLogger()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := start(ctx, log, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid port")
}

func TestStart(t *testing.T) {
	prefix := "PEERCALLS_"
	defer test.UnsetEnvPrefix(prefix)

	l, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	})
	require.NoError(t, err, "listener")
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	// os.Setenv(prefix+"BIND_ADDR", "127.0.0.1")
	os.Setenv(prefix+"BIND_PORT", strconv.Itoa(port))
	os.Setenv(prefix+"LOG", "-*")
	log := test.NewLogger()

	timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), 1*time.Second)
	ctx, cancel := context.WithCancel(timeoutCtx)

	defer cancelTimeout()
	defer cancel()

	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)
		err := start(ctx, log, []string{})
		errCh <- err
	}()

	var r *http.Response

	// Keep trying until the server finally starts.
	for i := 0; i < 30; i++ {
		r, err = http.Get("http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))

		if err != nil {
			time.Sleep(20 * time.Millisecond)

			continue
		}

		r.Body.Close()

		break
	}

	if assert.NoError(t, err) {
		assert.Equal(t, 200, r.StatusCode)
	}

	cancel()

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-timeoutCtx.Done():
		require.Fail(t, "timed out")
	}
}
