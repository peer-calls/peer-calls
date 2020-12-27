package test

import "github.com/peer-calls/peer-calls/server/logger"

func NewLogger() logger.Logger {
	return logger.NewFromEnv("PEERCALLS_LOG")
}
