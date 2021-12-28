package server

import (
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/message"
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

var ErrDuplicateClientID = errors.New("duplicate client id")
