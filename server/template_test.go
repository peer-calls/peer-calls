package server_test

import (
	"testing"

	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/stretchr/testify/assert"
)

func TestParseTemplates(t *testing.T) {
	t.Parallel()

	templates := server.ParseTemplates(embed.Templates)
	t1, ok := templates["index.html"]
	assert.Equal(t, true, ok)
	assert.NotNil(t, t1)
	t2, ok := templates["call.html"]
	assert.Equal(t, true, ok)
	assert.NotNil(t, t2)
}

func TestParseTemplates_noHTML(t *testing.T) {
	t.Parallel()

	templates := server.ParseTemplates(embed.Resources)
	t1, ok := templates["index.html"]
	assert.Equal(t, false, ok)
	assert.Nil(t, t1)
}
