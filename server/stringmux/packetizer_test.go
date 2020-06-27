package stringmux

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshal(t *testing.T) {
	data := []byte("somedata")

	payload, err := Marshal("mystream", data)
	require.NoError(t, err)

	assert.Equal(t, []byte{StringMuxByte, 8}, payload[0:2])

	streamID, data2, err := Unmarshal(payload)
	require.NoError(t, err)
	assert.Equal(t, "mystream", streamID)
	assert.Equal(t, data, data2)
}

func TestMarshal_StreamIDTooLong(t *testing.T) {
	streamID := string(make([]byte, 0xFF+1))
	_, err := Marshal(streamID, []byte{1, 2, 3})
	require.EqualError(t, err, "StreamID too large")
}

func TestUnmarshal_InvalidFirstTwoBytes(t *testing.T) {
	_, _, err := Unmarshal([]byte{123, 2, 'H', 'I', 1, 1})
	require.EqualError(t, err, "First byte should be 11001000")
}

func TestUnmarshal_InvalidHeader(t *testing.T) {
	_, _, err := Unmarshal([]byte{StringMuxByte})
	require.EqualError(t, err, "Header is too short")
}

func TestUnmarshal_LengthMismatch(t *testing.T) {
	_, _, err := Unmarshal([]byte{StringMuxByte, 3, 'T'})
	require.EqualError(t, err, "StreamID length mismatch")
}

func TestUnmarshal_OK(t *testing.T) {
	streamID, data, err := Unmarshal([]byte{StringMuxByte, 2, 'i', 'd', 'd', 'a', 't', 'a'})
	require.NoError(t, err)

	assert.Equal(t, "id", streamID)
	assert.Equal(t, "data", string(data))
}
