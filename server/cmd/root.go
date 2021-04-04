package cmd

import (
	"github.com/peer-calls/peer-calls/server/command"
)

func NewRootCommand(props Props) *command.Command {
	return command.New(command.Params{
		Name: "peer-calls",
		Desc: "Root peer-calls command",
		ArgsPreProcessor: command.ArgsProcessorFunc(func(c *command.Command, args []string) []string {
			if len(args) == 0 {
				return []string{"server"}
			}

			first := args[0]
			if len(first) > 0 && first[0] == '-' {
				return append([]string{"server"}, args...)
			}

			return args
		}),
		SubCommands: []*command.Command{
			newServerCmd(props),
		},
	})
}
