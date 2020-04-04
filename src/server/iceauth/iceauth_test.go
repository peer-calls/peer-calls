package iceauth_test

import (
	"testing"

	"github.com/jeremija/peer-calls/src/server/config"
	"github.com/jeremija/peer-calls/src/server/iceauth"
	"github.com/stretchr/testify/assert"
)

func TestGetICeServers(t *testing.T) {
	s1 := config.ICEServer{
		URLs: []string{"stun:"},
	}
	s2 := config.ICEServer{
		URLs:     []string{"turn:"},
		AuthType: config.AuthTypeSecret,
	}
	s2.AuthSecret.Username = "test"
	s2.AuthSecret.Secret = "sec"
	var servers []config.ICEServer
	servers = append(servers, s1, s2)

	result := iceauth.GetICEServers(servers)
	assert.Equal(t, 2, len(result))
	r1 := result[0]
	r2 := result[1]

	assert.Equal(t, s1.URLs, r1.URLs)
	assert.Equal(t, "", r1.Username)
	assert.Equal(t, "", r1.Credential)
	assert.Equal(t, s2.URLs, r2.URLs)
	assert.Regexp(t, "^[0-9]+:test$", r2.Username)
	assert.NotEmpty(t, r2.Credential)
}
