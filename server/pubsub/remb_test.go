package pubsub_test

import (
	"testing"

	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/pubsub"
	"github.com/stretchr/testify/assert"
)

func TestBitrateEstimator(t *testing.T) {
	t.Parallel()

	type feed struct {
		clientID identifiers.ClientID
		bitrate  float32
	}

	type expect struct {
		min, max, avg float32
		empty         bool
	}

	type test struct {
		name   string
		feed   *feed
		remove identifiers.ClientID
		expect *expect
	}

	tests := []test{
		{"initial state", nil, "", &expect{0, 0, 0, true}},
		{"feed a 45", &feed{"a", 45}, "", &expect{45, 45, 45, false}},
		{"feed a 50", &feed{"a", 50}, "", &expect{50, 50, 50, false}},
		{"feed b 60", &feed{"b", 60}, "", &expect{50, 60, 55, false}},
		{"remove b", nil, "b", &expect{50, 50, 50, false}},
		{"remove a", nil, "a", &expect{0, 0, 0, true}},
		{"feed a 45", &feed{"a", 45}, "", &expect{45, 45, 45, false}},
		{"feed b 50", &feed{"b", 50}, "", &expect{45, 50, 47.5, false}},
		{"feed c 30", &feed{"c", 30}, "", &expect{30, 50, 41.666668, false}},
		{"remove b", nil, "b", &expect{30, 45, 37.5, false}},
		{"remove a", nil, "a", &expect{30, 30, 30, false}},
		{"remove a again", nil, "a", &expect{30, 30, 30, false}},
		{"remove c", nil, "c", &expect{0, 0, 0, true}},
	}

	b := pubsub.NewBitrateEstimator()

	for _, test := range tests {
		if test.feed != nil {
			b.Feed(test.feed.clientID, test.feed.bitrate)
		}

		if test.remove != "" {
			b.RemoveClientBitrate(test.remove)
		}

		if test.expect != nil {
			assert.Equal(t, test.expect.empty, b.Empty(), "empty: %s", test.name)
			assert.Equal(t, test.expect.min, b.Min(), "min: %s", test.name)
			assert.Equal(t, test.expect.max, b.Max(), "max: %s", test.name)
			assert.Equal(t, test.expect.avg, b.Avg(), "avg: %s", test.name)
		}
	}
}
