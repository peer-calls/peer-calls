package logger

import "fmt"

// Level defines the logging level.
type Level int

const (
	// LevelUnknown is an unknown level.
	LevelUnknown Level = iota - 1

	// LevelDisabled means the logging is disabled and no messages will be logged.
	LevelDisabled

	// LevelError means only error messages will be logged.
	LevelError

	// LevelWarn means only warning, error and trace messages will be logged.
	LevelWarn

	// LevelInfo means only info, warning, error and trace messages will be
	// logged.
	LevelInfo

	// LevelDebug means debug, info, warning, error and trace messages will be
	// logged.
	LevelDebug

	// LevelTrace means all messages will be logged.
	LevelTrace
)

const (
	LevelDisabledString = "disabled"
	LevelErrorString    = "error"
	LevelWarnString     = "warn"
	LevelInfoString     = "info"
	LevelDebugString    = "debug"
	LevelTraceString    = "trace"
)

// String returns a string representation of Level.
func (l Level) String() string {
	switch l {
	case LevelError:
		return LevelErrorString
	case LevelWarn:
		return LevelWarnString
	case LevelInfo:
		return LevelInfoString
	case LevelDebug:
		return LevelDebugString
	case LevelTrace:
		return LevelTraceString
	case LevelDisabled:
		return LevelDisabledString
	case LevelUnknown:
		fallthrough
	default:
		return fmt.Sprintf("Unknown(%d)", l)
	}
}

func LevelFromString(str string) (Level, bool) {
	switch str {
	case LevelErrorString:
		return LevelError, true
	case LevelWarnString:
		return LevelWarn, true
	case LevelInfoString:
		return LevelInfo, true
	case LevelDebugString:
		return LevelDebug, true
	case LevelTraceString:
		return LevelTrace, true
	case LevelDisabledString:
		return LevelDisabled, true
	default:
		return LevelUnknown, false
	}
}

// LevelForNamespace implements Config. When a Level is passed as a config,
// all namespaces will have the same log level.
func (l Level) LevelForNamespace(_ string) Level {
	return l
}
