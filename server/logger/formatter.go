package logger

import (
	"fmt"
	"sort"
	"strings"
)

// Formatter defines the rules on how to format the logging context before
// transport. For example, a Formatter might prepare the context for writing to
// a log file, or serialize it to JSON before sending the bytes to transport.
type Formatter interface {
	// Format formats the logging context for transport.
	Format(message Message) ([]byte, error)
}

// StringFormatter is the default implementation of Formatter and it prepares
// the ctx for printing to stdout/stderr or a file.
type StringFormatter struct {
	params *StringFormatterParams
}

// StringFormatterParams are parameters for StringFormatter.
type StringFormatterParams struct {
	// DateLayout is the layout to be passed to time.Time.Format function for
	// formatting logging timestamp.
	DateLayout string

	// DisableContextKeySorting will not sort context keys before printing them.
	DisableContextKeySorting bool
}

// compile-time assertion that StringFormatter implements Formatter.
var _ Formatter = &StringFormatter{}

// NewStringFormatter creates a new instance of StringFormatter.
func NewStringFormatter(params StringFormatterParams) *StringFormatter {
	if params.DateLayout == "" {
		params.DateLayout = "2006-01-02T15:04:05.000000Z07:00"
	}

	return &StringFormatter{
		params: &params,
	}
}

// Format implements Formatter.
func (f *StringFormatter) Format(message Message) ([]byte, error) {
	// TODO maybe use sync.Pool for builders here.
	ctx := message.Ctx

	var keys []string

	if l := len(ctx); l > 0 {
		keys = make([]string, 0, l)

		for k := range ctx {
			keys = append(keys, k)
		}
	}

	if !f.params.DisableContextKeySorting {
		sort.Strings(keys)
	}

	var b strings.Builder

	for _, k := range keys {
		v := ctx[k]

		b.WriteString(" ")
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(fmt.Sprintf("%+v", v))
	}

	ret := fmt.Sprintf("%s %5s [%20s] %s%s\n",
		message.Timestamp.Format(f.params.DateLayout),
		message.Level,
		message.Namespace,
		strings.TrimRight(message.Body, "\n"),
		b.String(),
	)

	return []byte(ret), nil
}
