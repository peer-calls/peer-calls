package cli

import (
	"context"
	"fmt"

	"github.com/peer-calls/peer-calls/v4/server/command"
)

type versionHandler struct {
	props Props
}

func (v *versionHandler) Handle(ctx context.Context, args []string) error {
	fmt.Println("peer-calls", v.props.Version)

	return nil
}

func newVersionCmd(props Props) *command.Command {
	v := &versionHandler{props}

	return command.New(command.Params{
		Name:    "version",
		Desc:    "Show version information",
		Handler: v,
	})
}
