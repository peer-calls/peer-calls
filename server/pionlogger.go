package server

import "github.com/pion/logging"

type pionLogger struct {
	traceLogger Logger
	debugLogger Logger
	infoLogger  Logger
	warnLogger  Logger
	errorLogger Logger
}

type PionLoggerFactory struct {
	loggerFactory LoggerFactory
}

func NewPionLoggerFactory(loggerFactory LoggerFactory) *PionLoggerFactory {
	return &PionLoggerFactory{loggerFactory}
}

func (p PionLoggerFactory) NewLogger(subsystem string) logging.LeveledLogger {
	return &pionLogger{
		traceLogger: p.loggerFactory.GetLogger("pion:" + subsystem + ":trace"),
		debugLogger: p.loggerFactory.GetLogger("pion:" + subsystem + ":debug"),
		infoLogger:  p.loggerFactory.GetLogger("pion:" + subsystem + ":info"),
		warnLogger:  p.loggerFactory.GetLogger("pion:" + subsystem + ":warn"),
		errorLogger: p.loggerFactory.GetLogger("pion:" + subsystem + ":error"),
	}
}

func (p *pionLogger) Trace(msg string) {
	p.traceLogger.Println(msg)
}
func (p *pionLogger) Tracef(format string, args ...interface{}) {
	p.traceLogger.Printf(format, args...)
}
func (p *pionLogger) Debug(msg string) {
	p.debugLogger.Println(msg)
}
func (p *pionLogger) Debugf(format string, args ...interface{}) {
	p.debugLogger.Printf(format, args...)
}
func (p *pionLogger) Info(msg string) {
	p.infoLogger.Println(msg)
}
func (p *pionLogger) Infof(format string, args ...interface{}) {
	p.infoLogger.Printf(format, args...)
}
func (p *pionLogger) Warn(msg string) {
	p.warnLogger.Println(msg)
}
func (p *pionLogger) Warnf(format string, args ...interface{}) {
	p.warnLogger.Printf(format, args...)
}
func (p *pionLogger) Error(msg string) {
	p.errorLogger.Println(msg)
}
func (p *pionLogger) Errorf(format string, args ...interface{}) {
	p.errorLogger.Printf(format, args...)
}
