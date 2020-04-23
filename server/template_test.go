package server_test

import (
	"testing"

	"github.com/gobuffalo/packr"
	"github.com/peer-calls/peer-calls/server"
	"github.com/stretchr/testify/assert"
)

func TestParseTemplates(t *testing.T) {
	box := packr.NewBox("./templates")
	templates := server.ParseTemplates(box)
	t1, ok := templates["index.html"]
	assert.Equal(t, true, ok)
	assert.NotNil(t, t1)
	t2, ok := templates["call.html"]
	assert.Equal(t, true, ok)
	assert.NotNil(t, t2)
}

func TestParseTemplates_noHTML(t *testing.T) {
	box := packr.NewBox("../res")
	templates := server.ParseTemplates(box)
	t1, ok := templates["index.html"]
	assert.Equal(t, false, ok)
	assert.Nil(t, t1)
}
