package logger2_test

import (
	"testing"

	"github.com/peer-calls/peer-calls/server/logger2"
	"github.com/stretchr/testify/assert"
)

func TestLevel_String(t *testing.T) {
	t.Parallel()

	type testCase struct {
		level      logger2.Level
		wantString string
	}

	testCases := []testCase{
		{logger2.LevelError, "error"},
		{logger2.LevelWarn, "warn"},
		{logger2.LevelInfo, "info"},
		{logger2.LevelDebug, "debug"},
		{logger2.LevelTrace, "trace"},
		{logger2.LevelDisabled, "disabled"},
		{logger2.LevelUnknown, "Unknown(-1)"},
		{logger2.Level(-3), "Unknown(-3)"},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.wantString, tc.level.String())
	}
}

func TestLevelFromString(t *testing.T) {
	t.Parallel()

	type testCase struct {
		levelName string
		wantLevel logger2.Level
		wantOK    bool
	}

	testCases := []testCase{
		{"error", logger2.LevelError, true},
		{"warn", logger2.LevelWarn, true},
		{"info", logger2.LevelInfo, true},
		{"debug", logger2.LevelDebug, true},
		{"trace", logger2.LevelTrace, true},
		{"disabled", logger2.LevelDisabled, true},
		{"something-else", logger2.LevelUnknown, false},
	}

	for _, tc := range testCases {
		level, ok := logger2.LevelFromString(tc.levelName)

		assert.Equal(t, tc.wantLevel, level)
		assert.Equal(t, tc.wantOK, ok)
	}
}
