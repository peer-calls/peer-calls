package cmd

import (
	"context"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
)

type Props struct {
	Log     logger.Logger
	Version string
	Args    []string
}

func Exec(ctx context.Context, props Props) error {
	cmd := NewRootCommand(props)
	err := cmd.Exec(ctx, props.Args)

	return errors.Trace(err)
}
