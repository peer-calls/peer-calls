package server_test

import (
	"os"
	"strings"
	"testing"

	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/peer-calls/peer-calls/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadConfig(t *testing.T) {
	c, err := server.ReadConfig([]string{})
	assert.Nil(t, err, "error reading config")
	assert.Equal(t, 2, len(c.ICEServers))
	assert.Equal(t, []string{"stun:stun.l.google.com:19302"}, c.ICEServers[0].URLs)
	assert.Equal(t, []string{"stun:global.stun.twilio.com:3478?transport=udp"}, c.ICEServers[1].URLs)
	assert.Equal(t, server.NetworkTypeMesh, c.Network.Type)
	assert.Equal(t, server.StoreTypeMemory, c.Store.Type)
}

func TestReadConfigFiles(t *testing.T) {
	var c server.Config
	err := server.ReadConfigFiles([]string{"config_example.yml"}, &c)
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, "/test", c.BaseURL)
	assert.Equal(t, "test.pem", c.TLS.Cert)
	assert.Equal(t, "test.key", c.TLS.Key)
	assert.Equal(t, server.StoreTypeRedis, c.Store.Type)
	assert.Equal(t, "localhost", c.Store.Redis.Host)
	assert.Equal(t, 6379, c.Store.Redis.Port)
	assert.Equal(t, "peercalls", c.Store.Redis.Prefix)
	assert.Equal(t, 1, len(c.ICEServers))
	ice := c.ICEServers[0]
	assert.Equal(t, []string{"stun:stun.l.google.com:19302"}, ice.URLs)
	assert.Equal(t, server.AuthTypeSecret, ice.AuthType)
	assert.Equal(t, "test_user", ice.AuthSecret.Username)
	assert.Equal(t, "test_secret", ice.AuthSecret.Secret)
	assert.Equal(t, []string(nil), c.Network.SFU.Interfaces)
	assert.Equal(t, server.NetworkTypeSFU, c.Network.Type)
	assert.Equal(t, uint16(9000), c.Network.SFU.UDP.PortMin)
	assert.Equal(t, uint16(9010), c.Network.SFU.UDP.PortMax)
}

func TestReadConfigFiles_Error(t *testing.T) {
	var c server.Config
	err := server.ReadConfigFiles([]string{"config_missing.yml"}, &c)
	require.NotNil(t, err, "error should be defined")
	assert.Regexp(t, "no such file", err.Error())
}

func TestReadYAML_error(t *testing.T) {
	yaml := "gfakjhglakjhlakdhgl"
	reader := strings.NewReader(yaml)
	var c server.Config
	err := server.ReadConfigYAML(reader, &c)
	require.NotNil(t, err, "err should be defined")
	assert.Regexp(t, "decode yaml", err.Error())
}

func TestReadFromEnv(t *testing.T) {
	prefix := "PEERCALLSTEST_"
	defer test.UnsetEnvPrefix(prefix)
	os.Setenv(prefix+"BASE_URL", "/test")
	os.Setenv(prefix+"TLS_CERT", "test.pem")
	os.Setenv(prefix+"TLS_KEY", "test.key")
	os.Setenv(prefix+"STORE_TYPE", "redis")
	os.Setenv(prefix+"STORE_REDIS_HOST", "localhost")
	os.Setenv(prefix+"STORE_REDIS_PORT", "6379")
	os.Setenv(prefix+"STORE_REDIS_PREFIX", "peercalls")
	os.Setenv(prefix+"ICE_SERVER_URLS", "stun:stun.l.google.com:19302,stuns:stun.l.google.com:19302")
	os.Setenv(prefix+"ICE_SERVER_AUTH_TYPE", "secret")
	os.Setenv(prefix+"ICE_SERVER_USERNAME", "test_user")
	os.Setenv(prefix+"ICE_SERVER_SECRET", "test_secret")
	os.Setenv(prefix+"NETWORK_TYPE", "sfu")
	os.Setenv(prefix+"NETWORK_SFU_PROTOCOLS", "tcp6,udp4")
	os.Setenv(prefix+"NETWORK_SFU_TCP_LISTEN_PORT", "8443")
	os.Setenv(prefix+"NETWORK_SFU_INTERFACES", "a,b")
	os.Setenv(prefix+"NETWORK_SFU_JITTER_BUFFER", "true")
	os.Setenv(prefix+"NETWORK_SFU_UDP_PORT_MIN", "9000")
	os.Setenv(prefix+"NETWORK_SFU_UDP_PORT_MAX", "9010")
	os.Setenv(prefix+"PROMETHEUS_ACCESS_TOKEN", "at1234")
	os.Setenv(prefix+"NETWORK_SFU_TRANSPORT_NODES", "127.0.0.1:3005,127.0.0.1:3006")
	os.Setenv(prefix+"NETWORK_SFU_TRANSPORT_LISTEN_ADDR", "127.0.0.1:3004")
	var c server.Config
	server.ReadConfigFromEnv(prefix, &c)
	assert.Equal(t, "/test", c.BaseURL)
	assert.Equal(t, "test.pem", c.TLS.Cert)
	assert.Equal(t, "test.key", c.TLS.Key)
	assert.Equal(t, server.StoreTypeRedis, c.Store.Type)
	assert.Equal(t, "localhost", c.Store.Redis.Host)
	assert.Equal(t, 6379, c.Store.Redis.Port)
	assert.Equal(t, "peercalls", c.Store.Redis.Prefix)
	assert.Equal(t, 1, len(c.ICEServers))
	assert.Equal(t, []server.ICEServer{
		{
			URLs: []string{
				"stun:stun.l.google.com:19302",
				"stuns:stun.l.google.com:19302",
			},
			AuthType: server.AuthTypeSecret,
			AuthSecret: struct {
				Username string `yaml:"username"`
				Secret   string `yaml:"secret"`
			}{"test_user", "test_secret"},
		},
	}, c.ICEServers)
	assert.Equal(t, []string{"tcp6", "udp4"}, c.Network.SFU.Protocols)
	assert.Equal(t, 8443, c.Network.SFU.TCPListenPort)
	assert.Equal(t, server.NetworkType("sfu"), c.Network.Type)
	assert.Equal(t, []string{"a", "b"}, c.Network.SFU.Interfaces)
	assert.Equal(t, true, c.Network.SFU.JitterBuffer)
	assert.Equal(t, uint16(9000), c.Network.SFU.UDP.PortMin)
	assert.Equal(t, uint16(9010), c.Network.SFU.UDP.PortMax)
	assert.Equal(t, "at1234", c.Prometheus.AccessToken)
	assert.Equal(t, "127.0.0.1:3004", c.Network.SFU.Transport.ListenAddr)
	assert.Equal(t, []string{"127.0.0.1:3005", "127.0.0.1:3006"}, c.Network.SFU.Transport.Nodes)

	t.Run("disable default ICE servers", func(t *testing.T) {
		prefix := "PEERCALLSTEST_"
		defer test.UnsetEnvPrefix(prefix)

		var c server.Config

		server.InitConfig(&c)

		assert.Equal(t, []server.ICEServer{
			{
				URLs: []string{
					"stun:stun.l.google.com:19302",
				},
			},
			{
				URLs: []string{
					"stun:global.stun.twilio.com:3478?transport=udp",
				},
			},
		}, c.ICEServers)

		os.Setenv(prefix+"ICE_SERVER_URLS", "")
		server.ReadConfigFromEnv(prefix, &c)

		assert.Equal(t, []server.ICEServer{}, c.ICEServers)
	})
}
