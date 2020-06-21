package stringmux

import (
	"encoding/binary"
	"fmt"
)

func Marshal(streamID string, data []byte) []byte {
	/*
	 *         0                   1                   2                   3
	 *         0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	 *        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	 * header |P E E R C A L L| StreamID len  |        Stream ID ....         |
	 *        +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
	 *        |                           StreamID                            |
	 *        |                              ...                              |
	 *        +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
	 *        |                              Data                             |
	 *        |                              ...                              |
	 *        +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
	 */

	lenStreamID := len(streamID)
	result := make([]byte, 0, 16+lenStreamID+len(data))

	result = append(result, []byte("PEERCALL")...)

	binary.BigEndian.PutUint32(result[8:16], uint32(lenStreamID))

	copy(result[16:], data)

	return result
}

func Unmarshal(data []byte) (string, []byte, error) {
	if len(data) < 16 {
		return "", nil, fmt.Errorf("Header of data to unmarshal is too short")
	}

	if string(data[0:8]) != "PEERCALL" {
		return "", nil, fmt.Errorf("First header should be PEERCALL")
	}

	lenStreamID := binary.BigEndian.Uint32(data[8:16])

	if len(data) < int(16+lenStreamID) {
		return "", nil, fmt.Errorf("Canont extract streamID from data")
	}

	startStreamID := uint32(16)
	endStreamID := startStreamID + lenStreamID
	streamID := string(data[startStreamID:endStreamID])
	result := data[endStreamID:]

	return streamID, result, nil
}
