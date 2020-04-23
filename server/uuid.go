package server

import "github.com/google/uuid"

var defaultBaseNEncoder = NewBaseNEncoder(AlphabetBase62)

func NewUUIDBase62() string {
	value := uuid.New()
	return defaultBaseNEncoder.Encode(value[:])
}
