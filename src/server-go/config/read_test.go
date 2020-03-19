package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/jeremija/peer-calls/src/server-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRead(t *testing.T) {
	_, err := config.Read([]string{})
	assert.Nil(t, err, "error reading config")
}

func TestDefaults(t *testing.T) {
	var c config.Config
	config.Defaults(&c)
	assert.Equal(t, "/", c.BaseURL)
}

func TestReadFiles(t *testing.T) {
	var c config.Config
	err := config.ReadFiles([]string{"config_example.yml"}, &c)
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, "/test", c.BaseURL)
	assert.Equal(t, "test.pem", c.TLS.Cert)
	assert.Equal(t, "test.key", c.TLS.Key)
	assert.Equal(t, config.StoreTypeRedis, c.Store.Type)
	assert.Equal(t, "localhost", c.Store.Redis.Host)
	assert.Equal(t, 6379, c.Store.Redis.Port)
	assert.Equal(t, "peercalls", c.Store.Redis.Prefix)
	assert.Equal(t, 1, len(c.ICEServers))
	ice := c.ICEServers[0]
	assert.Equal(t, []string{"stun:stun.l.google.com:19302"}, ice.URLs)
	assert.Equal(t, config.AuthTypeSecret, ice.AuthType)
	assert.Equal(t, "test_user", ice.AuthSecret.Username)
	assert.Equal(t, "test_secret", ice.AuthSecret.Secret)
}

func TestReadFiles_error(t *testing.T) {
	var c config.Config
	err := config.ReadFiles([]string{"config_missing.yml"}, &c)
	require.NotNil(t, err, "error should be defined")
	assert.Regexp(t, "no such file", err.Error())
}

func TestReadYAML_error(t *testing.T) {
	yaml := "gfakjhglakjhlakdhgl"
	reader := strings.NewReader(yaml)
	var c config.Config
	err := config.ReadYAML(reader, &c)
	require.NotNil(t, err, "err should be defined")
	assert.Regexp(t, "Error parsing YAML", err.Error())
}

func TestReadFromEnv(t *testing.T) {
	prefix := "PEERCALLSTEST_"
	defer os.Unsetenv(prefix)
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
	var c config.Config
	config.ReadEnv(prefix, &c)
	assert.Equal(t, "/test", c.BaseURL)
	assert.Equal(t, "test.pem", c.TLS.Cert)
	assert.Equal(t, "test.key", c.TLS.Key)
	assert.Equal(t, config.StoreTypeRedis, c.Store.Type)
	assert.Equal(t, "localhost", c.Store.Redis.Host)
	assert.Equal(t, 6379, c.Store.Redis.Port)
	assert.Equal(t, "peercalls", c.Store.Redis.Prefix)
	assert.Equal(t, 1, len(c.ICEServers))
	ice := c.ICEServers[0]
	assert.Equal(t, []string{
		"stun:stun.l.google.com:19302",
		"stuns:stun.l.google.com:19302",
	}, ice.URLs)
	assert.Equal(t, config.AuthTypeSecret, ice.AuthType)
	assert.Equal(t, "test_user", ice.AuthSecret.Username)
	assert.Equal(t, "test_secret", ice.AuthSecret.Secret)
}
