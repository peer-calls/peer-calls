package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// WriterLogger is a logger that writes to io.Writer when it is enabled.
type WriterLogger struct {
	name    string
	out     io.Writer
	outMu   sync.Mutex
	Enabled bool
}

// Logger is an interface for logger
type Logger interface {
	// Printf formats a message and writes to output. If logger is not enabled,
	// the message will not be formatted.
	Printf(message string, values ...interface{})
	// Println writes all values similar to fmt.Println. If logger is not enabled,
	// the message will not be formatted
	Println(values ...interface{})
}

// LoggerTimeFormat is the time format used by loggers in this package
var LoggerTimeFormat = "2006-01-02T15:04:05.000000Z07:00"

// NewWriterLogger creates a new logger
func NewWriterLogger(name string, out io.Writer, enabled bool) *WriterLogger {
	return &WriterLogger{name: name, out: out, Enabled: enabled}
}

// Printf implements Logger#Printf func.
func (l *WriterLogger) Printf(message string, values ...interface{}) {
	if l.Enabled {
		l.printf(message, values...)
	}
}

// Println implements Logger#Println func.
func (l *WriterLogger) Println(values ...interface{}) {
	if l.Enabled {
		l.println(values...)
	}
}

func (l *WriterLogger) printf(message string, values ...interface{}) {
	l.outMu.Lock()
	defer l.outMu.Unlock()
	date := time.Now().Format(LoggerTimeFormat)
	l.out.Write([]byte(date + fmt.Sprintf(" [%15s] ", l.name) + fmt.Sprintf(message+"\n", values...)))
}

func (l *WriterLogger) println(values ...interface{}) {
	l.outMu.Lock()
	defer l.outMu.Unlock()
	date := time.Now().Format(LoggerTimeFormat)
	l.out.Write([]byte(date + fmt.Sprintf(" [%15s] ", l.name) + fmt.Sprintln(values...)))
}

// Factory creates new loggers. Only one logger with a specific name
// will be created.
type Factory struct {
	out            io.Writer
	loggers        map[string]*WriterLogger
	defaultEnabled []string
	loggersMu      sync.Mutex
}

// NewFactory creates a new logger factory. The enabled slice can be used
// to set the default enabled loggers. Enabled string can contain strings
// delimited with colon character, and can use wildcards. For example, if a
// we have a logger with name `myproject:a:b`, it can be enabled by setting
// the enabled string to `myproject:a:b`, or `myproject:*` or `myproject:*:b`.
// To disable a logger, add a minus to the beginning of the name. For example,
// to enable all loggers but one use: `-myproject:a:b,*`.
func NewFactory(out io.Writer, enabled []string) *Factory {
	return &Factory{
		out:            out,
		loggers:        map[string]*WriterLogger{},
		defaultEnabled: enabled,
	}
}

// NewFactoryFromEnv creates a new Factory and reads the enabled
// loggers from a comma-delimited environment variable.
func NewFactoryFromEnv(prefix string, out io.Writer) *Factory {
	log := os.Getenv(prefix + "LOG")
	var enabled []string
	if len(log) > 0 {
		enabled = strings.Split(log, ",")
	}
	return NewFactory(out, enabled)
}

// SetDefaultEnabled sets enabled loggers if the Factory has been
// initialized with no loggers.
func (l *Factory) SetDefaultEnabled(names []string) {
	if len(l.defaultEnabled) == 0 {
		l.defaultEnabled = names
		for name, logger := range l.loggers {
			if !logger.Enabled {
				logger.Enabled = l.isEnabled(name)
			}
		}
	}
}

func split(name string) (parts []string) {
	if len(name) > 0 {
		parts = strings.Split(name, ":")
	}
	return
}

func partsMatch(parts []string, enabledParts []string) bool {
	isLastWildcard := false
	for i, part := range parts {
		if len(enabledParts) <= i {
			return isLastWildcard
		}

		isLastWildcard = false
		enabledPart := enabledParts[i]

		if enabledPart == part {
			continue
		}

		if enabledPart == "*" {
			isLastWildcard = true
			continue
		}

		return false
	}

	return true
}

func (l *Factory) isEnabled(name string) bool {
	parts := split(name)

	for _, enabledName := range l.defaultEnabled {
		isEnabled := true

		if strings.HasPrefix(enabledName, "-") {
			enabledName = enabledName[1:]
			isEnabled = false
		}

		enabledParts := split(enabledName)

		if partsMatch(parts, enabledParts) {
			return isEnabled
		}
	}

	return false
}

// GetLogger creates or retrieves an existing logger with name. It is thread
// safe.
func (l *Factory) GetLogger(name string) Logger {
	l.loggersMu.Lock()
	defer l.loggersMu.Unlock()
	logger, ok := l.loggers[name]
	if !ok {
		enabled := l.isEnabled(name)
		logger = NewWriterLogger(name, l.out, enabled)
		l.loggers[name] = logger
	}
	return logger
}
