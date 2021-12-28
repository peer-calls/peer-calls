package server

import (
	"encoding/json"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/message"
)

type Serializer interface {
	Serialize(message.Message) ([]byte, error)
}

type Deserializer interface {
	Deserialize([]byte) (message.Message, error)
}

type ByteSerializer struct{}

func (s ByteSerializer) Serialize(m message.Message) ([]byte, error) {
	b, err := json.Marshal(m)
	return b, errors.Annotate(err, "serialize")
}

func (s ByteSerializer) Deserialize(data []byte) (msg message.Message, err error) {
	err = json.Unmarshal(data, &msg)
	return msg, errors.Annotate(err, "deserialize")
}
