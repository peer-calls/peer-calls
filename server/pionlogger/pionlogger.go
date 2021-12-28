package pionlogger

import (
	"fmt"

	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/pion/logging"
)

type pionLogger struct {
	log logger.Logger
}

type Factory struct {
	log logger.Logger
}

func NewFactory(log logger.Logger) *Factory {
	log = log.WithNamespaceAppended("pion")
	return &Factory{log}
}

func (p Factory) NewLogger(subsystem string) logging.LeveledLogger {
	return &pionLogger{
		log: p.log.WithNamespaceAppended(subsystem),
	}
}

func (p *pionLogger) Trace(msg string) {
	p.log.Trace(msg, nil)
}

func (p *pionLogger) Tracef(format string, args ...interface{}) {
	if p.log.IsLevelEnabled(logger.LevelTrace) {
		p.Trace(fmt.Sprintf(format, args...))
	}
}

func (p *pionLogger) Debug(msg string) {
	p.log.Debug(msg, nil)
}

func (p *pionLogger) Debugf(format string, args ...interface{}) {
	if p.log.IsLevelEnabled(logger.LevelDebug) {
		p.Debug(fmt.Sprintf(format, args...))
	}
}

func (p *pionLogger) Info(msg string) {
	p.log.Info(msg, nil)
}

func (p *pionLogger) Infof(format string, args ...interface{}) {
	if p.log.IsLevelEnabled(logger.LevelInfo) {
		p.Info(fmt.Sprintf(format, args...))
	}
}

func (p *pionLogger) Warn(msg string) {
	p.log.Warn(msg, nil)
}

func (p *pionLogger) Warnf(format string, args ...interface{}) {
	if p.log.IsLevelEnabled(logger.LevelWarn) {
		p.Warn(fmt.Sprintf(format, args...))
	}
}

func (p *pionLogger) Error(msg string) {
	p.log.Error(msg, nil, nil)
}

func (p *pionLogger) Errorf(format string, args ...interface{}) {
	if p.log.IsLevelEnabled(logger.LevelError) {
		p.Error(fmt.Sprintf(format, args...))
	}
}
