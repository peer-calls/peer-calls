package server

import "github.com/peer-calls/peer-calls/server/transport"

type (
	Transport         = transport.Transport
	MetadataTransport = transport.MetadataTransport
	MediaTransport    = transport.MediaTransport
	DataTransport     = transport.DataTransport
	Closable          = transport.Closable
)
