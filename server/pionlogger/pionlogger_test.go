package pionlogger

import (
	"fmt"
	"strings"
	"testing"

	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/stretchr/testify/assert"
)

func TestPionLogger(t *testing.T) {
	t.Parallel()

	type entry struct {
		level   string
		message string
		args    []interface{}
	}

	type testCase struct {
		entries   []entry
		subsystem string
		want      string
	}

	testCases := []testCase{
		{
			entries: []entry{
				{"debug", "test", nil},
			},
		},
	}

	var b strings.Builder

	for i, tc := range testCases {
		descr := fmt.Sprintf("test case: %d", i)

		log := logger.New().WithWriter(&b).WithConfig(
			logger.NewConfig(logger.ConfigMap{
				"pion": logger.LevelTrace,
			}),
		).WithFormatter(logger.NewStringFormatter(logger.StringFormatterParams{
			DisableContextKeySorting: false,
			DateLayout:               "-",
		}))

		plf := NewFactory(log)

		subsystem := tc.subsystem
		if subsystem == "" {
			subsystem = "pion:test"
		}

		pionLogger := plf.NewLogger(subsystem)

		for _, entry := range tc.entries {
			switch entry.level {
			case "trace":
				pionLogger.Trace(entry.message)
			case "tracef":
				pionLogger.Tracef(entry.message, entry.args...)
			case "debug":
				pionLogger.Debug(entry.message)
			case "debugf":
				pionLogger.Debugf(entry.message, entry.args...)
			case "info":
				pionLogger.Info(entry.message)
			case "infof":
				pionLogger.Infof(entry.message, entry.args...)
			case "warn":
				pionLogger.Warn(entry.message)
			case "warnf":
				pionLogger.Warnf(entry.message, entry.args...)
			case "error":
				pionLogger.Error(entry.message)
			case "errorf":
				pionLogger.Errorf(entry.message, entry.args...)
			default:
				panic(fmt.Sprintf("unknown level: %s", entry.level))
			}
		}

		assert.Equal(t, tc.want, b.String(), descr)
	}
}
