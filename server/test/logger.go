package test

import (
	"github.com/peer-calls/peer-calls/v4/server/logformatter"
	"github.com/peer-calls/peer-calls/v4/server/logger"
)

func NewLogger() logger.Logger {
	return logger.NewFromEnv("PEERCALLS_LOG").WithFormatter(logformatter.New())
}
