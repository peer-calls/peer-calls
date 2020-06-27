package stringmux

import (
	"fmt"
)

const (
	StringMuxByte  uint8 = 0b11001000
	MaxLenStreamID int   = 0xFF
)

func Marshal(streamID string, data []byte) ([]byte, error) {
	/*
	 *         0                   1                   2                   3
	 *         0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	 *        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	 * header | StringMuxByte | StreamID len  |   StreamID ...                |
	 *        +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
	 *        |                           StreamID                            |
	 *        |                              ...                              |
	 *        +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
	 *        |                              Data                             |
	 *        |                              ...                              |
	 *        +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
	 */

	lenStreamID := len(streamID)
	if lenStreamID > MaxLenStreamID {
		return nil, fmt.Errorf("StreamID too large")
	}

	result := make([]byte, 2+lenStreamID+len(data))

	offset := 0

	result[offset] = StringMuxByte
	offset++

	result[offset] = uint8(lenStreamID)
	offset++

	copy(result[offset:offset+lenStreamID], streamID)
	offset += lenStreamID

	copy(result[offset:], data)

	return result, nil
}

func Unmarshal(data []byte) (string, []byte, error) {
	if len(data) < 2 {
		return "", nil, fmt.Errorf("Header is too short")
	}

	offset := 0

	if data[0] != StringMuxByte {
		return "", nil, fmt.Errorf("First byte should be %b", StringMuxByte)
	}
	offset++

	lenStreamID := int(data[offset])
	offset++

	if len(data) < offset+lenStreamID {
		return "", nil, fmt.Errorf("StreamID length mismatch")
	}

	streamID := string(data[offset : offset+lenStreamID])
	offset += lenStreamID

	result := data[offset:]

	return streamID, result, nil
}
