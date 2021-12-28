package logger_test

import (
	"fmt"
	"testing"

	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/stretchr/testify/assert"
)

func TestWildcardNode(t *testing.T) {
	t.Parallel()

	assert.Equal(t, logger.Config(nil), logger.NewConfig(nil))

	configMap := logger.ConfigMap{
		"a":                logger.Level(1),
		"a:b":              logger.Level(2),
		"a:b:*":            logger.Level(3),
		"a:*:c":            logger.Level(4),
		"*:d":              logger.Level(5),
		"a:b:c:d:e:f":      logger.Level(6),
		"":                 logger.Level(7),
		"aa:**:cc":         logger.Level(8),
		"**:double:left":   logger.Level(9),
		"double:right:**":  logger.Level(10),
		"**:both:sides:**": logger.Level(11),
	}

	config := logger.NewConfig(configMap)

	type testCase struct {
		namespace string
		wantLevel logger.Level
	}

	testCases := []testCase{
		{"", 7},
		{"something:else", 7},
		{"something:else:d", 7},
		{"h:g:d", 7},
		{"a:b:c", 3},
		{"a:b", 2},
		{"a", 1},
		{"a:x:c", 4},
		{"a:x:y:c", 7},
		{"a:x:y:z:c", 7},
		{"a:b:c:d:e:f", 6},
		{"aa:cc", 8},
		{"aa:xx:cc", 8},
		{"aa:xx:yy:cc", 8},
		{"aa:xx:yy:zz:cc", 8},
		{"double:left", 9},
		{"xx:double:left", 9},
		{"xx:yy:double:left", 9},
		{"xx:yy:zz:double:left", 9},
		{"double:right", 10},
		{"double:right:xx", 10},
		{"double:right:xx:yy", 10},
		{"double:right:xx:yy:zz", 10},
		{"both:sides", 11},
		{"both:sides:xx", 11},
		{"both:sides:xx:yy", 11},
		{"xx:both:sides", 11},
		{"xx:yy:both:sides", 11},
		{"xx:yy:both:sides:zz", 11},
		{"xx:yy:both:sides:ww:zz", 11},
	}

	for i, tc := range testCases {
		descr := fmt.Sprintf("test case: %d, namespace: %q", i, tc.namespace)

		assert.Equal(t, tc.wantLevel, config.LevelForNamespace(tc.namespace), descr)
	}
}
