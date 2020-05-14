package main

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/peer-calls/peer-calls/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartMissingConfig(t *testing.T) {
	prefix := "PEERCALLS_"
	defer test.UnsetEnvPrefix(prefix)
	os.Setenv(prefix+"BIND_PORT", "0")
	os.Setenv(prefix+"LOG", "-*")
	_, stop, errCh := start([]string{"-c", "/missing/file.yml"})
	assert.Nil(t, stop)
	err := <-errCh
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Error reading config")
}

func TestStartWrongPort(t *testing.T) {
	prefix := "PEERCALLS_"
	defer test.UnsetEnvPrefix(prefix)
	os.Setenv(prefix+"BIND_PORT", "100000")
	os.Setenv(prefix+"LOG", "-*")
	_, stop, errCh := start([]string{})
	assert.Nil(t, stop)
	err := <-errCh
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid port")
}

func TestStart(t *testing.T) {
	prefix := "PEERCALLS_"
	defer test.UnsetEnvPrefix(prefix)
	os.Setenv(prefix+"BIND_PORT", "0")
	os.Setenv(prefix+"LOG", "-*")
	addr, stop, errCh := start([]string{})
	r, err := http.Get("http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(addr.Port)))
	assert.NoError(t, err)
	assert.Equal(t, 200, r.StatusCode)
	stop()
	err = <-errCh
	assert.NoError(t, err)
}
