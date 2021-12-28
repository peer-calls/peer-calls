package cli

import (
	"context"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/peer-calls/peer-calls/v4/server/logger"
)

type Props struct {
	Log     logger.Logger
	Version string
	Args    []string
	Embed   server.Embed
}

func Exec(ctx context.Context, props Props) error {
	cmd := NewRootCommand(props)
	err := cmd.Exec(ctx, props.Args)

	return errors.Trace(err)
}
