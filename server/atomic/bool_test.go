package atomic_test

import (
	"testing"

	"github.com/peer-calls/peer-calls/v4/server/atomic"
	"github.com/stretchr/testify/assert"
)

func TestBool(t *testing.T) {
	var b atomic.Bool

	assert.False(t, b.Get())
	assert.False(t, b.CompareAndSwap(false))
	assert.False(t, b.Get())

	b.Set(false)
	assert.False(t, b.Get())

	b.Set(true)
	assert.True(t, b.Get())

	assert.False(t, b.CompareAndSwap(true))
	assert.True(t, b.Get())

	assert.True(t, b.CompareAndSwap(false))
	assert.False(t, b.Get())

	assert.True(t, b.CompareAndSwap(true))
	assert.True(t, b.Get())

	assert.False(t, b.CompareAndSwap(true))
	assert.True(t, b.Get())

	assert.True(t, b.CompareAndSwap(false))
	assert.False(t, b.Get())
}
