package stringmux

import (
	"encoding/binary"
	"fmt"
	"math"
)

func Marshal(streamID string, data []byte) ([]byte, error) {
	/*
	 *         0                   1                   2                   3
	 *         0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	 *        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	 * header |        P      |       C       |         Stream ID len         |
	 *        +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
	 *        |                           StreamID                            |
	 *        |                              ...                              |
	 *        +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
	 *        |                              Data                             |
	 *        |                              ...                              |
	 *        +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
	 */

	lenStreamID := len(streamID)
	if lenStreamID > math.MaxUint16 {
		return nil, fmt.Errorf("StreamID too large")
	}

	result := make([]byte, 4+lenStreamID+len(data))

	offset := 0

	result[offset] = 'P'
	offset++

	result[offset] = 'C'
	offset++

	binary.BigEndian.PutUint16(result[offset:offset+2], uint16(lenStreamID))
	offset += 2

	copy(result[offset:offset+lenStreamID], streamID)
	offset += lenStreamID

	copy(result[offset:], data)

	return result, nil
}

func Unmarshal(data []byte) (string, []byte, error) {
	if len(data) < 4 {
		return "", nil, fmt.Errorf("Header is too short")
	}

	offset := 0

	if string(data[0:2]) != "PC" {
		return "", nil, fmt.Errorf("First two bytes should be 'PC'")
	}
	offset += 2

	lenStreamID := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2

	if len(data) < offset+int(lenStreamID) {
		return "", nil, fmt.Errorf("StreamID length mismatch")
	}

	streamID := string(data[offset : offset+int(lenStreamID)])
	offset += int(lenStreamID)

	result := data[offset:]

	return streamID, result, nil
}
