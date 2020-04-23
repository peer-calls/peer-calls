package server

type ClientWriter interface {
	ID() string
	Write(message Message) error
	Metadata() string
	SetMetadata(metadata string)
}

type Adapter interface {
	Add(client ClientWriter) error
	Remove(clientID string) error
	Broadcast(msg Message) error
	Metadata(clientID string) (string, bool)
	SetMetadata(clientID string, metadata string) bool
	Emit(clientID string, msg Message) error
	Clients() (map[string]string, error)
	Size() (int, error)
	Close() error
}
