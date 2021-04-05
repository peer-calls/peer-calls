package server

import (
	"github.com/peer-calls/peer-calls/server/identifiers"
	"github.com/peer-calls/peer-calls/server/message"
)

type ClientWriter interface {
	ID() identifiers.ClientID
	Write(msg message.Message) error
	Metadata() string
	SetMetadata(metadata string)
}

type Adapter interface {
	Add(client ClientWriter) error
	Remove(clientID identifiers.ClientID) error
	Broadcast(msg message.Message) error
	Metadata(clientID identifiers.ClientID) (string, bool)
	SetMetadata(clientID identifiers.ClientID, metadata string) bool
	Emit(clientID identifiers.ClientID, msg message.Message) error
	Clients() (map[identifiers.ClientID]string, error)
	Size() (int, error)
	Close() error
}
