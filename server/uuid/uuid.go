package uuid

import (
	"github.com/google/uuid"
	"github.com/peer-calls/peer-calls/v4/server/basen"
)

var defaultBaseNEncoder = basen.NewBaseNEncoder(basen.AlphabetBase62)

func New() string {
	value := uuid.New()

	return defaultBaseNEncoder.Encode(value[:])
}
