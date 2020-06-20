package test

import (
	"os"

	"github.com/peer-calls/peer-calls/server/logger"
)

func NewLoggerFactory() logger.LoggerFactory {
	return logger.NewFactoryFromEnv("PEERCALLS_", os.Stdout)
}
