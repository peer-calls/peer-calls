package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type WriterLogger struct {
	name    string
	out     io.Writer
	outMu   sync.Mutex
	Enabled bool
}

type Logger interface {
	Printf(message string, values ...interface{})
	Println(values ...interface{})
}

var WriterLoggerTimeFormat = "2006-01-02T15:04:05.000000Z07:00"

func NewWriterLogger(name string, out io.Writer, enabled bool) *WriterLogger {
	return &WriterLogger{name: name, out: out, Enabled: enabled}
}

func (l *WriterLogger) Printf(message string, values ...interface{}) {
	if l.Enabled {
		l.printf(message, values...)
	}
}

func (l *WriterLogger) Println(values ...interface{}) {
	if l.Enabled {
		l.println(values...)
	}
}

func (l *WriterLogger) printf(message string, values ...interface{}) {
	l.outMu.Lock()
	defer l.outMu.Unlock()
	date := time.Now().Format(WriterLoggerTimeFormat)
	l.out.Write([]byte(date + fmt.Sprintf(" [%15s] ", l.name) + fmt.Sprintf(message+"\n", values...)))
}

func (l *WriterLogger) println(values ...interface{}) {
	l.outMu.Lock()
	defer l.outMu.Unlock()
	date := time.Now().Format(WriterLoggerTimeFormat)
	l.out.Write([]byte(date + fmt.Sprintf(" [%15s] ", l.name) + fmt.Sprintln(values...)))
}

type LoggerFactory struct {
	out            io.Writer
	loggers        map[string]*WriterLogger
	defaultEnabled []string
	loggersMu      sync.Mutex
}

func NewLoggerFactory(out io.Writer, enabled []string) *LoggerFactory {
	return &LoggerFactory{
		out:            out,
		loggers:        map[string]*WriterLogger{},
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

func (l *LoggerFactory) GetLogger(name string) Logger {
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
