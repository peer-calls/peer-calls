package server_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/peer-calls/peer-calls/server"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

var handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))
})

func TestServerStarter_HTTP(t *testing.T) {
	defer goleak.VerifyNone(t)
	addr := net.JoinHostPort("127.0.0.1", "0")
	l, err := net.Listen("tcp", addr)
	port := l.Addr().(*net.TCPAddr).Port
	require.Nil(t, err, "error listening to: %s", addr)
	s := server.NewStartStopper(server.ServerParams{}, handler)
	go s.Start(l)
	defer s.Stop()
	var c http.Client
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	r, err := http.NewRequest("GET", url, nil)
	require.Nil(t, err, "error creating new request")
	res, err := c.Do(r)
	require.Nil(t, err, "error executing request")
	body, err := ioutil.ReadAll(res.Body)
	require.Nil(t, err, "error reading body")
	require.Equal(t, []byte("hello"), body)
}

func TestServerStarter_HTTPS(t *testing.T) {
	defer goleak.VerifyNone(t)
	addr := net.JoinHostPort("127.0.0.1", "0")
	l, err := net.Listen("tcp", addr)
	port := l.Addr().(*net.TCPAddr).Port
	require.Nil(t, err, "error listening to: %s", addr)
	params := server.ServerParams{
		TLSCertFile: "../config/cert.example.pem",
		TLSKeyFile:  "../config/cert.example.key",
	}
	s := server.NewStartStopper(params, handler)
	go s.Start(l)
	defer s.Stop()
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
