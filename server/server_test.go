package server_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

var handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))
})

func TestServer_HTTP(t *testing.T) {
	defer goleak.VerifyNone(t)
	addr := net.JoinHostPort("127.0.0.1", "0")
	l, err := net.Listen("tcp", addr)
	port := l.Addr().(*net.TCPAddr).Port
	require.Nil(t, err, "error listening to: %s", addr)
	s := server.New(server.Params{}, handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go s.Start(ctx, l)

	var c http.Client
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	r, err := http.NewRequest("GET", url, nil)
	require.Nil(t, err, "error creating new request")
	res, err := c.Do(r)
	require.Nil(t, err, "error executing request")
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	require.Nil(t, err, "error reading body")
	require.Equal(t, []byte("hello"), body)
}

func TestServer_HTTPS(t *testing.T) {
	defer goleak.VerifyNone(t)
	addr := net.JoinHostPort("127.0.0.1", "0")
	l, err := net.Listen("tcp", addr)
	port := l.Addr().(*net.TCPAddr).Port
	require.Nil(t, err, "error listening to: %s", addr)
	params := server.Params{
		TLSCertFile: "../config/cert.example.pem",
		TLSKeyFile:  "../config/cert.example.key",
	}
	s := server.New(params, handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go s.Start(ctx, l)

	var c http.Client
	c.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	url := fmt.Sprintf("https://127.0.0.1:%d", port)
	r, err := http.NewRequest("GET", url, nil)
	require.Nil(t, err, "error creating new request")
	res, err := c.Do(r)
	require.Nil(t, err, "error executing request")
	body, err := ioutil.ReadAll(res.Body)
	require.Nil(t, err, "error reading body")
	require.Equal(t, []byte("hello"), body)
}
