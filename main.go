package main

import (
	"context"
	"os"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/cli"
	"github.com/peer-calls/peer-calls/server/logformatter"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/multierr"
	"github.com/spf13/pflag"
)

const gitDescribe string = "v0.0.0"

func start(ctx context.Context, log logger.Logger, args []string) error {
	err := cli.Exec(ctx, cli.Props{
		Log:     log,
		Version: gitDescribe,
		Args:    args,
	})

	return errors.Trace(err)
}

func main() {
	log := logger.New().
		WithConfig(
			logger.NewConfig(logger.ConfigMap{
				"**:sdp":          logger.LevelError,
				"**:ws":           logger.LevelError,
				"**:nack":         logger.LevelError,
				"**:signaller:**": logger.LevelError,
				"**:pion:**":      logger.LevelWarn,
				"**:pubsub":       logger.LevelTrace,
				"**:factory":      logger.LevelTrace,
				"**:sdp:**":       logger.LevelInfo,
				"":                logger.LevelInfo,
			}),
		).
		WithConfig(logger.NewConfigFromString(os.Getenv("PEERCALLS_LOG"))). // FIXME use wildcards here
		WithFormatter(logformatter.New()).
		WithNamespaceAppended("main")

	err := start(context.Background(), log, os.Args[1:])

	if multierr.Is(err, pflag.ErrHelp) {
		os.Exit(1)
	} else if err != nil {
		log.Error("Command error", errors.Trace(err), nil)
		os.Exit(1)
	}
}
