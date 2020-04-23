package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	name    string
	out     io.Writer
	outMu   sync.Mutex
	Enabled bool
}

var TimeFormat = "2006-01-02T15:04:05.000000Z07:00"

func NewLogger(name string, out io.Writer, enabled bool) *Logger {
	return &Logger{name: name, out: out, Enabled: enabled}
}

func (l *Logger) Printf(message string, values ...interface{}) {
	if l.Enabled {
		l.printf(message, values...)
	}
}

func (l *Logger) Println(values ...interface{}) {
	if l.Enabled {
		l.println(values...)
	}
}

func (l *Logger) printf(message string, values ...interface{}) {
	l.outMu.Lock()
	defer l.outMu.Unlock()
	date := time.Now().Format(TimeFormat)
	l.out.Write([]byte(date + fmt.Sprintf(" [%15s] ", l.name) + fmt.Sprintf(message+"\n", values...)))
}

func (l *Logger) println(values ...interface{}) {
	l.outMu.Lock()
	defer l.outMu.Unlock()
	date := time.Now().Format(TimeFormat)
	l.out.Write([]byte(date + fmt.Sprintf(" [%15s] ", l.name) + fmt.Sprintln(values...)))
}

type LoggerFactory struct {
	out            io.Writer
	loggers        map[string]*Logger
	defaultEnabled []string
	loggersMu      sync.Mutex
}

func NewLoggerFactory(out io.Writer, enabled []string) *LoggerFactory {
	return &LoggerFactory{
		out:            out,
		loggers:        map[string]*Logger{},
		defaultEnabled: enabled,
	}
}

func NewLoggerFactoryFromEnv(prefix string, out io.Writer) *LoggerFactory {
	log := os.Getenv(prefix + "LOG")
	var enabled []string
	if len(log) > 0 {
		enabled = strings.Split(log, ",")
	}
	return NewLoggerFactory(out, enabled)
}

// Sets default enabled loggers if none have been read from environment
func (l *LoggerFactory) SetDefaultEnabled(names []string) {
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

func (l *LoggerFactory) isEnabled(name string) bool {
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

func (l *LoggerFactory) GetLogger(name string) *Logger {
	l.loggersMu.Lock()
	defer l.loggersMu.Unlock()
	logger, ok := l.loggers[name]
	if !ok {
		enabled := l.isEnabled(name)
		logger = NewLogger(name, l.out, enabled)
		l.loggers[name] = logger
	}
	return logger
}

var defaultLoggerFactory = NewLoggerFactoryFromEnv("PEERCALLS_", os.Stderr)
var GetLogger = defaultLoggerFactory.GetLogger
var SetDefaultEnabled = defaultLoggerFactory.SetDefaultEnabled
