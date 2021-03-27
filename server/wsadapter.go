package server

import "github.com/peer-calls/peer-calls/server/identifiers"

type ClientWriter interface {
	ID() identifiers.ClientID
	Write(message Message) error
	Metadata() string
	SetMetadata(metadata string)
}

type Adapter interface {
	Add(client ClientWriter) error
	Remove(clientID identifiers.ClientID) error
	Broadcast(msg Message) error
	Metadata(clientID identifiers.ClientID) (string, bool)
	SetMetadata(clientID identifiers.ClientID, metadata string) bool
	Emit(clientID identifiers.ClientID, msg Message) error
	Clients() (map[identifiers.ClientID]string, error)
	Size() (int, error)
	Close() error
}
