package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNTP(t *testing.T) {
	t1 := time.Date(1995, 11, 10, 11, 33, 25, 125_000_000, time.UTC)
	t2 := time.Date(1995, 11, 10, 11, 33, 36, 5_000_000, time.UTC)

	assert.Equal(t, NTPTime(0xb44d_b705_2000_0000), NewNTPTime(t1))
	assert.Equal(t, NTPTime(0xb44d_b710_0147_ae14), NewNTPTime(t2))

	assert.Equal(t, t1.String(), NewNTPTime(t1).Time().String())
	assert.Equal(t, "1995-11-10 11:33:36.004999999 +0000 UTC", NewNTPTime(t2).Time().String())
}
