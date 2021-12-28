package command_test

import (
	"context"
	"testing"

	"github.com/peer-calls/peer-calls/v4/server/command"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestCommand_NoArgsAndNoSubcommands(t *testing.T) {
	var got []string

	cmd := command.New(command.Params{
		Name: "root",
		Desc: "Root is the root command",
		Handler: command.HandlerFunc(
			func(ctx context.Context, args []string) error {
				got = args

				return nil
			},
		),
		SubCommands: nil,
	})

	args := []string{"a", "b", "c"}

	err := cmd.Exec(context.Background(), args)
	assert.NoError(t, err, "error exec")

	assert.Equal(t, args, got, "expected to receive same arguments")
}

func TestCommand_SimpleArgs(t *testing.T) {
	var got []string
	var config string

	cmd := command.New(command.Params{
		Name: "root",
		Desc: "Root is the root command",
		FlagRegistry: command.FlagRegistryFunc(func(_ *command.Command, flags *pflag.FlagSet) {
			flags.StringVarP(&config, "config", "c", "", "config to use")
		}),
		Handler: command.HandlerFunc(func(ctx context.Context, args []string) error {
			got = args

			return nil
		}),
		SubCommands: nil,
	})

	args := []string{"-c", "myconfig.yaml", "a", "-b", "c"}

	err := cmd.Exec(context.Background(), args)
	assert.NoError(t, err, "error exec")

	assert.Equal(t, config, "myconfig.yaml")

	assert.Equal(t, args[2:], got, "expected only unused arguments")
}

func TestCommand_SubCommand(t *testing.T) {
	var got []string
	var config string
	var file string

	cmd := command.New(command.Params{
		Name: "root",
		Desc: "Root is the root command",
		FlagRegistry: command.FlagRegistryFunc(func(_ *command.Command, flags *pflag.FlagSet) {
			flags.StringVarP(&config, "config", "c", "", "config to use")
		}),
		Handler: command.HandlerFunc(func(ctx context.Context, args []string) error {
			got = args

			return nil
		}),
		SubCommands: []*command.Command{
			command.New(command.Params{
				Name: "sub1",
				Desc: "sub desc",
				FlagRegistry: command.FlagRegistryFunc(func(_ *command.Command, flags *pflag.FlagSet) {
					flags.StringVarP(&file, "file", "f", "", "file to use")
				}),
				Handler: command.HandlerFunc(func(ctx context.Context, args []string) error {
					got = args

					return nil
				}),
				SubCommands: nil,
			}),
		},
	})

	type test struct {
		name       string
		exec       []string
		wantConfig string
		wantFile   string
		wantArgs   []string
		wantErr    string
	}

	tests := []test{
		{
			name:       "call sub1",
			exec:       []string{"--config", "config.yaml", "sub1", "--file", "myfile", "test"},
			wantConfig: "config.yaml",
			wantFile:   "myfile",
			wantArgs:   []string{"test"},
		},
		{
			name:       "call sub1 with --",
			exec:       []string{"--config", "config.yaml", "--", "sub1", "--file", "myfile", "test"},
			wantConfig: "config.yaml",
			wantFile:   "myfile",
			wantArgs:   []string{"test"},
		},
		{
			name:       "command not found",
			exec:       []string{"--config", "config.yaml", "sub2", "--file", "myfile", "test"},
			wantConfig: "config.yaml",
			wantFile:   "",
			wantErr:    "command: sub2: command not found",
			wantArgs:   []string{"sub2", "--file", "myfile", "test"},
		},
		{
			name:     "parsing error in root command",
			exec:     []string{"--invalid", "config.yaml", "sub1", "--file", "myfile", "test"},
			wantErr:  "parse args for command: root: unknown flag: --invalid",
			wantArgs: nil,
		},
		{
			name:       "parsing error in subcommand",
			exec:       []string{"--config", "config.yaml", "sub1", "--invalid", "myfile", "test"},
			wantConfig: "config.yaml",
			wantErr:    "parse args for command: sub1: unknown flag: --invalid",
			wantArgs:   []string{"sub1", "--invalid", "myfile", "test"},
		},
	}

	for _, test := range tests {
		got = nil
		config = ""
		file = ""

		err := cmd.Exec(context.Background(), test.exec)
		if test.wantErr != "" {
			if assert.Error(t, err, "wantErr: %s", test.name) {
				assert.Equal(t, test.wantErr, err.Error(), "wantErr: %s", test.name)
			}
		} else {
			assert.NoError(t, err, "wantErr: %s", test.name)
		}

		assert.Equal(t, test.wantConfig, config, "wantConfig: %s", test.name)
		assert.Equal(t, test.wantFile, file, "wantFile: %s", test.name)
		assert.Equal(t, test.wantArgs, got, "wantArgs: %s", test.name)
	}
}
