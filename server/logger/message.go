package logger

import "time"

type Message struct {
	// Timestamp contains the time of the message.
	Timestamp time.Time

	// Namespace is the full namespace of the Logger this message was sent to.
	Namespace string

	// Level is the log level of the message.
	Level Level

	// Body has the message contents.
	Body string

	// Ctx is the message context.
	Ctx Ctx
}
