package stringmux

import (
	"github.com/juju/errors"
)

const (
	StringMuxByte  uint8 = 0b11001000
	MaxLenStreamID int   = 0xFF
)

var (
	ErrStreamIDTooLarge = errors.New("stream id too large")
	ErrInvalidHeader    = errors.New("invalid first byte")
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
		return nil, errors.Annotate(ErrStreamIDTooLarge, "marshal")
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
	headerSize := 2

	if l := len(data); l < headerSize {
		return "", nil, errors.Annotatef(ErrInvalidHeader, "size: %d", l)
	}

	offset := 0

	if data[0] != StringMuxByte {
		return "", nil, errors.Annotatef(ErrInvalidHeader, "expected: %+v, got: %+v", StringMuxByte, data[0])
	}
	offset++

	lenStreamID := int(data[offset])
	offset++

	if len(data) < offset+lenStreamID {
		return "", nil, errors.Annotatef(ErrInvalidHeader, "expected: %d, got: %d", offset+lenStreamID, len(data))
	}

	streamID := string(data[offset : offset+lenStreamID])
	offset += lenStreamID

	result := data[offset:]

	return streamID, result, nil
}
