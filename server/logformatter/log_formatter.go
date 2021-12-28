package logformatter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/peer-calls/peer-calls/v4/server/logger"
)

// LogFormatter adds special peer-calls specific formatting for console output.
type LogFormatter struct {
}

func New() *LogFormatter {
	return &LogFormatter{}
}

var _ logger.Formatter = &LogFormatter{}

func (f *LogFormatter) Format(message logger.Message) ([]byte, error) {
	ctx := message.Ctx

	var keys []string

	if l := len(ctx); l > 0 {
		keys = make([]string, 0, l)

		for k := range ctx {
			keys = append(keys, k)
		}
	}

	sort.Strings(keys)

	var b strings.Builder

	clientIDKey := "client_id"

	var clientID string

	for _, k := range keys {
		v := ctx[k]

		if k == clientIDKey {
			clientID = fmt.Sprintf("%s", v)

			continue
		}

		b.WriteString(" ")
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(fmt.Sprintf("%+v", v))
	}

	var ret string

	namespace := message.Namespace

	if l := 20; len(namespace) > l {
		namespace = namespace[len(namespace)-l:]
	}

	timeLayout := "2006-01-02T15:04:05.000000Z07:00"

	if clientID != "" {
		ret = fmt.Sprintf("%s %5s [%20s] [%s] %s%s\n",
			message.Timestamp.Format(timeLayout),
			message.Level,
			namespace,
			clientID,
			strings.TrimRight(message.Body, "\n"),
			b.String(),
		)
	} else {
		ret = fmt.Sprintf("%s %5s [%20s] %s%s\n",
			message.Timestamp.Format(timeLayout),
			message.Level,
			namespace,
			strings.TrimRight(message.Body, "\n"),
			b.String(),
		)
	}

	return []byte(ret), nil
}
