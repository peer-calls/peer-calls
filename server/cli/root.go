package cli

import (
	"github.com/peer-calls/peer-calls/v4/server/command"
)

func NewRootCommand(props Props) *command.Command {
	return command.New(command.Params{
		Name: "peer-calls",
		Desc: "Peer Calls is a distributed conferencing solution.",
		ArgsPreProcessor: command.ArgsProcessorFunc(func(c *command.Command, args []string) []string {
			for _, arg := range args {
				if len(arg) > 0 && arg[0] != '-' {
					break
				}

				if arg == "-h" || arg == "--help" {
					return args
				}
			}

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
			newPlayCmd(props),
			newVersionCmd(props),
		},
	})
}
