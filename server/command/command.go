package command

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/juju/errors"
	"github.com/spf13/pflag"
)

var ErrCommandNotFound = errors.New("command not found")

// Handler is command line handler.
type Handler interface {
	// Handle receives the context and the arguments leftover from parsing.
	// When the first return value is non-nil and the error is nil, the value
	// will be passed as an argument to the subcommand.
	Handle(ctx context.Context, args []string) error
}

// HandlerFunc defines a functional implementation of Handler.
type HandlerFunc func(ctx context.Context, args []string) error

// Handle implements Handler interface.
func (h HandlerFunc) Handle(ctx context.Context, args []string) error {
	return h(ctx, args)
}

// FlagRegistry contains optional methods for parsing CLI arguments.
type FlagRegistry interface {
	// RegisterFlags can be implemented to register custom flags.
	RegisterFlags(cmd *Command, flags *pflag.FlagSet)
}

// FlagRegistryFunc defines a functional implementation of FlagRegistry.
type FlagRegistryFunc func(cmd *Command, flags *pflag.FlagSet)

// FlagRegistryFunc implements FlagRegistry.
func (f FlagRegistryFunc) RegisterFlags(cmd *Command, flags *pflag.FlagSet) {
	f(cmd, flags)
}

type ArgsProcessor interface {
	ProcessArgs(c *Command, args []string) []string
}

// ArgsProcessorFunc defines a functional implementation of ArgsProcessor.
type ArgsProcessorFunc func(cmd *Command, args []string) []string

// ArgsProcessorFunc implements ArgsProcessor.
func (f ArgsProcessorFunc) ProcessArgs(cmd *Command, args []string) []string {
	return f(cmd, args)
}

type Command struct {
	params      Params
	subCommands map[string]*Command
	writer      io.Writer
}

type Params struct {
	Name              string
	Desc              string
	ArgsPreProcessor  ArgsProcessor
	ArgsPostProcessor ArgsProcessor
	FlagRegistry      FlagRegistry
	Handler           Handler
	SubCommands       []*Command
}

func New(params Params) *Command {
	var subCommands map[string]*Command

	if len(params.SubCommands) > 0 {
		subCommands = make(map[string]*Command, len(params.SubCommands))

		for _, cmd := range params.SubCommands {
			subCommands[cmd.Name()] = cmd
		}
	}

	c := &Command{
		params:      params,
		subCommands: subCommands,
	}

	c.SetWriter(os.Stderr)

	return c
}

func (c *Command) SetWriter(w io.Writer) {
	c.writer = w

	for _, s := range c.params.SubCommands {
		s.SetWriter(w)
	}
}

func (c Command) Name() string {
	return c.params.Name
}

func (c Command) Desc() string {
	return c.params.Desc
}

func (c Command) Usage(flags *pflag.FlagSet) {
	var b bytes.Buffer

	b.WriteString("Usage: ")

	flagUsages := flags.FlagUsages()

	hasOptions := flagUsages != ""
	hasSubCommands := len(c.params.SubCommands) > 0

	b.WriteString(c.params.Name)

	if hasOptions {
		b.WriteString(" [OPTIONS]")
	}

	if hasSubCommands {
		b.WriteString(" [COMMAND] [ARG...]")
	}

	b.WriteString("\n")
	b.WriteString(c.params.Desc)
	b.WriteString("\n")

	if hasOptions {
		b.WriteString("\nOptions:\n")
		b.WriteString(flags.FlagUsages())
		b.WriteString("\n")
	}

	if hasSubCommands {
		b.WriteString("\nCommands:\n")

		maxLen := 12
		for _, s := range c.params.SubCommands {
			if ll := len(s.Name()); ll > maxLen {
				maxLen = ll
			}
		}

		for _, s := range c.params.SubCommands {
			b.WriteString(fmt.Sprintf("  %-*s %s\n", maxLen, s.Name(), s.Desc()))
		}

		b.WriteString("\n")
	}

	_, _ = b.WriteTo(c.writer)
}

func (c *Command) Exec(ctx context.Context, args []string) error {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	flags := pflag.NewFlagSet(c.Name(), pflag.ContinueOnError)

	flags.SetOutput(c.writer)

	flags.Usage = func() {
		c.Usage(flags)
	}

	if c.params.ArgsPreProcessor != nil {
		args = c.params.ArgsPreProcessor.ProcessArgs(c, args)
	}

	// Need to set this to allow easier processing of subcommands.
	flags.SetInterspersed(false)

	if c.params.FlagRegistry != nil {
		c.params.FlagRegistry.RegisterFlags(c, flags)
	}

	err := flags.Parse(args)
	if err != nil {
		return errors.Annotatef(err, "parse args for command: %s", c.params.Name)
	}

	args = flags.Args()

	if c.params.Handler != nil {
		err = c.params.Handler.Handle(ctx, args)
		if err != nil {
			return errors.Trace(err)
		}
	}

	if c.params.ArgsPostProcessor != nil {
		args = c.params.ArgsPostProcessor.ProcessArgs(c, args)
	}

	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}

	if len(args) > 0 && len(c.subCommands) > 0 {
		subName := args[0]
		subCommand, ok := c.subCommands[subName]
		if !ok {
			return errors.Annotatef(ErrCommandNotFound, "command: %s", subName)
		}

		err := subCommand.Exec(ctx, args[1:])
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
