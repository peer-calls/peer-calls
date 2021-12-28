package main

import (
	"context"
	"embed"
	"io/fs"
	"os"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/peer-calls/peer-calls/v4/server/cli"
	"github.com/peer-calls/peer-calls/v4/server/logformatter"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/multierr"
	"github.com/spf13/pflag"
)

//nolint:gochecknoglobals
//go:embed server/templates/*.html
var templatesFS embed.FS

//nolint:gochecknoglobals
//go:embed build/*.js build/style.css
var staticFS embed.FS

//nolint:gochecknoglobals
//go:embed res/*
var resourcesFS embed.FS

// GitDescribe contains the version information.
// nolint:gochecknoglobals
var GitDescribe = "v0.0.0"

func mustSub(dir fs.FS, path string) fs.FS {
	fs, err := fs.Sub(dir, path)
	if err != nil {
		panic(err)
	}

	return fs
}

func start(ctx context.Context, log logger.Logger, args []string) error {
	err := cli.Exec(ctx, cli.Props{
		Log:     log,
		Version: GitDescribe,
		Args:    args,
		Embed: server.Embed{
			Resources: mustSub(resourcesFS, "res"),
			Templates: mustSub(templatesFS, "server/templates"),
			Static:    mustSub(staticFS, "build"),
		},
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
				"":                logger.LevelInfo,
			}),
		).
		WithConfig(logger.NewConfigFromString(os.Getenv("PEERCALLS_LOG"))).
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
