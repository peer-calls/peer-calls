package basen

import "github.com/google/uuid"

var defaultEncoder = NewEncoder(AlphabetBase62)

func NewUUIDBase62() string {
	value := uuid.New()
	return defaultEncoder.Encode(value[:])
}
