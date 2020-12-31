package servertransport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchRTP(t *testing.T) {
	for i := 0; i < 128; i++ {
		assert.False(t, MatchRTP([]byte{byte(i)}))
	}
	for i := 128; i < 192; i++ {
		assert.True(t, MatchRTP([]byte{byte(i)}))
	}

	for i := 192; i < 255; i++ {
		assert.False(t, MatchRTP([]byte{byte(i)}))
	}
}

func TestMatchRTCP(t *testing.T) {
	// too short
	assert.False(t, MatchRTCP([]byte{128, 192, 0}))

	for i := 0; i < 128; i++ {
		for j := 0; j < 255; j++ {
			assert.False(t, MatchRTCP([]byte{byte(i), byte(j), 0, 0}))
		}
	}
	for i := 128; i < 192; i++ {
		for j := 0; j < 192; j++ {
			assert.False(t, MatchRTCP([]byte{byte(i), byte(j), 0, 0}))
		}
		for j := 192; j < 224; j++ {
			assert.True(t, MatchRTCP([]byte{byte(i), byte(j), 0, 0}))
		}
		for j := 224; j < 255; j++ {
			assert.False(t, MatchRTCP([]byte{byte(i), byte(j), 0, 0}))
		}
	}
	for i := 192; i < 255; i++ {
		for j := 0; j < 255; j++ {
			assert.False(t, MatchRTCP([]byte{byte(i), byte(j), 0, 0}))
		}
	}
}
