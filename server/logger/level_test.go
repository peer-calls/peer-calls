package logger_test

import (
	"testing"

	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/stretchr/testify/assert"
)

func TestLevel_String(t *testing.T) {
	t.Parallel()

	type testCase struct {
		level      logger.Level
		wantString string
	}

	testCases := []testCase{
		{logger.LevelError, "error"},
		{logger.LevelWarn, "warn"},
		{logger.LevelInfo, "info"},
		{logger.LevelDebug, "debug"},
		{logger.LevelTrace, "trace"},
		{logger.LevelDisabled, "disabled"},
		{logger.LevelUnknown, "Unknown(-1)"},
		{logger.Level(-3), "Unknown(-3)"},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.wantString, tc.level.String())
	}
}

func TestLevelFromString(t *testing.T) {
	t.Parallel()

	type testCase struct {
		levelName string
		wantLevel logger.Level
		wantOK    bool
	}

	testCases := []testCase{
		{"error", logger.LevelError, true},
		{"warn", logger.LevelWarn, true},
		{"info", logger.LevelInfo, true},
		{"debug", logger.LevelDebug, true},
		{"trace", logger.LevelTrace, true},
		{"disabled", logger.LevelDisabled, true},
		{"something-else", logger.LevelUnknown, false},
	}

	for _, tc := range testCases {
		level, ok := logger.LevelFromString(tc.levelName)

		assert.Equal(t, tc.wantLevel, level)
		assert.Equal(t, tc.wantOK, ok)
	}
}
