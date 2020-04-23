package server

import "github.com/peer-calls/peer-calls/server/logger"

type Logger = logger.Logger

type LoggerFactory interface {
	GetLogger(name string) Logger
}
