package logger

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Logger is an interface for logger.
type Logger interface {
	Factory

	// Level returns the current logger's level.
	Level() Level

	Namespace() string

	// IsLevelEnabled returns true when Level is enabled, false otherwise.
	IsLevelEnabled(level Level) bool

	// Trace adds a log entry with level trace.
	Trace(message string, ctx Ctx) (int, error)

	// Debug adds a log entry with level debug.
	Debug(message string, ctx Ctx) (int, error)

	// Info adds a log entry with level info.
	Info(message string, ctx Ctx) (int, error)

	// Warn adds a log entry with level warn.
	Warn(message string, ctx Ctx) (int, error)

	// Error adds a log entry with level error.
	Error(message string, err error, ctx Ctx) (int, error)
}

type Factory interface {
	// Ctx returns the current logger's context.
	Ctx() Ctx

	// WithCtx returns a new Logger with context appended to existing context.
	WithCtx(Ctx) Logger

	// WithFormatter returns a new Logger with formatter set.
	WithFormatter(Formatter) Logger

	// WithWriter returns a new Logger with writer set.
	WithWriter(io.Writer) Logger

	// WithNamespace returns a new Logger with namespace set.
	WithNamespace(namespace string) Logger

	// WithNamespaceAppended returns a new Logger with namespace appended.
	WithNamespaceAppended(namespace string) Logger

	// WithConfig returns a new Logger with config set.
	WithConfig(config Config) Logger
}

// logger is a logger that writes to io.Writer when it is enabled.
type logger struct {
	config    Config
	ctx       Ctx
	formatter Formatter
	namespace string
	writer    io.Writer
}

// New returns a new Logger with default StringFormatter. Be sure to call

// WithConfig to set the required levels for different namespaces.
func New() Logger {
	return &logger{
		config:    LevelDisabled,
		ctx:       nil,
		formatter: NewStringFormatter(StringFormatterParams{}),
		namespace: "",
		writer:    os.Stderr,
	}
}

func NewFromEnv(key string) Logger {
	envConfig := os.Getenv(key)

	return New().WithConfig(NewConfigMapFromString(envConfig))
}

// compile-time assertion that logger implements Logger.
var _ Logger = &logger{}

// Ctx implements Logger.
func (l *logger) Ctx() Ctx {
	return l.ctx
}

// WithCtx implements Logger.
func (l *logger) WithCtx(ctx Ctx) Logger {
	return &logger{
		config:    l.config,
		ctx:       l.ctx.WithCtx(ctx),
		formatter: l.formatter,
		namespace: l.namespace,
		writer:    l.writer,
	}
}

// WithFormatter implements Logger.
func (l *logger) WithFormatter(formatter Formatter) Logger {
	return &logger{
		config:    l.config,
		ctx:       l.ctx,
		formatter: formatter,
		namespace: l.namespace,
		writer:    l.writer,
	}
}

// WithWriter implements Logger.
func (l *logger) WithWriter(writer io.Writer) Logger {
	return &logger{
		config:    l.config,
		ctx:       l.ctx,
		formatter: l.formatter,
		namespace: l.namespace,
		writer:    writer,
	}
}

// WithNamespace implements Logger.
func (l *logger) WithNamespace(namespace string) Logger {
	return &logger{
		config:    l.config,
		ctx:       l.ctx,
		formatter: l.formatter,
		namespace: namespace,
		writer:    l.writer,
	}
}

// WithNamespaceAppended implements Logger.
func (l *logger) WithNamespaceAppended(newNamespace string) Logger {
	oldNamespace := l.namespace

	if oldNamespace != "" {
		newNamespace = fmt.Sprintf("%s:%s", oldNamespace, newNamespace)
	}

	return l.WithNamespace(newNamespace)
}

// WithConfig implements Logger.
func (l *logger) WithConfig(config Config) Logger {
	if config == nil {
		return l
	}

	return &logger{
		config:    config,
		ctx:       l.ctx,
		formatter: l.formatter,
		namespace: l.namespace,
		writer:    l.writer,
	}
}

// Level implements Logger.
func (l *logger) Namespace() string {
	return l.namespace
}

// Level implements Logger.
func (l *logger) Level() Level {
	return l.config.LevelForNamespace(l.namespace)
}

// Trace implements Logger.
func (l *logger) Trace(message string, ctx Ctx) (int, error) {
	i, err := l.log(time.Now(), LevelTrace, message, ctx)

	return i, err
}

// Debug implements Logger.
func (l *logger) Debug(message string, ctx Ctx) (int, error) {
	i, err := l.log(time.Now(), LevelDebug, message, ctx)

	return i, err
}

// Info implements Logger.
func (l *logger) Info(message string, ctx Ctx) (int, error) {
	i, err := l.log(time.Now(), LevelInfo, message, ctx)

	return i, err
}

// Warn implements Logger.
func (l *logger) Warn(message string, ctx Ctx) (int, error) {
	i, err := l.log(time.Now(), LevelWarn, message, ctx)

	return i, err
}

// Error implements Logger.
func (l *logger) Error(message string, err error, ctx Ctx) (int, error) {
	if err != nil {
		if message != "" {
			message = fmt.Sprintf("%s: %+v", message, err)
		} else {
			message = fmt.Sprintf("%+v", err)
		}
	}

	i, err := l.log(time.Now(), LevelError, message, ctx)

	return i, err
}

// IsLevelEnabled implements Logger.
func (l *logger) IsLevelEnabled(level Level) bool {
	configuredLevel := l.Level()

	return configuredLevel > 0 && level <= configuredLevel
}

func (l *logger) log(ts time.Time, level Level, message string, ctx Ctx) (int, error) {
	if !l.IsLevelEnabled(level) {
		return 0, nil
	}

	formatted, err := l.formatter.Format(Message{
		Timestamp: ts,
		Namespace: l.namespace,
		Level:     level,
		Body:      message,
		Ctx:       l.ctx.WithCtx(ctx),
	})
	if err != nil {
		return 0, fmt.Errorf("log format error: %w", err)
	}

	i, err := l.writer.Write(formatted)
	if err != nil {
		return i, fmt.Errorf("log write error: %w", err)
	}

	return i, nil
}
