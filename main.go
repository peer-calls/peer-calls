package main

import (
	"context"
	"fmt"
	"os"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/cmd"
	"github.com/peer-calls/peer-calls/server/logformatter"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/multierr"
	"github.com/spf13/pflag"
)

const gitDescribe string = "v0.0.0"

func start(ctx context.Context, args []string) error {
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

	err := cmd.Exec(ctx, cmd.Props{
		Log:     log,
		Version: gitDescribe,
		Args:    args,
	})

	return errors.Trace(err)
}

func main() {
	err := start(context.Background(), os.Args[1:])

	if multierr.Is(err, pflag.ErrHelp) {
		os.Exit(1)
	} else if err != nil {
		fmt.Printf("Command error: %+v\n", errors.Trace(err))
		os.Exit(1)
	}
}
