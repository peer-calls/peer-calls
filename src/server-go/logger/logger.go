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
	defaultEnabled map[string]struct{}
	loggersMu      sync.Mutex
}

func NewLoggerFactory(out io.Writer, enabled map[string]struct{}) *LoggerFactory {
	return &LoggerFactory{
		out:            out,
		loggers:        map[string]*Logger{},
		defaultEnabled: enabled,
	}
}

func NewLoggerFactoryFromEnv(prefix string, out io.Writer) *LoggerFactory {
	log := os.Getenv(prefix + "LOG")
	if log == "" {
		log = "*"
	}
	enabled := strings.Split(log, ",")
	defaultEnabled := map[string]struct{}{}
	for _, name := range enabled {
		defaultEnabled[name] = struct{}{}
	}
	return NewLoggerFactory(out, defaultEnabled)
}

func (l *LoggerFactory) GetLogger(name string) *Logger {
	l.loggersMu.Lock()
	defer l.loggersMu.Unlock()
	logger, ok := l.loggers[name]
	if !ok {
		_, enabled := l.defaultEnabled[name]
		_, allEnabled := l.defaultEnabled["*"]
		logger = NewLogger(name, l.out, enabled || allEnabled)
		l.loggers[name] = logger
	}
	return logger
}

var GetLogger = NewLoggerFactoryFromEnv("", os.Stderr).GetLogger
