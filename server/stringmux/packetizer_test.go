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

	assert.Equal(t, []byte{'P', 'C', 0, 8}, payload[0:4])

	streamID, data2, err := Unmarshal(payload)
	require.NoError(t, err)
	assert.Equal(t, "mystream", streamID)
	assert.Equal(t, data, data2)
}

func TestMarshal_StreamIDTooLong(t *testing.T) {
	streamID := string(make([]byte, 65535+1))
	_, err := Marshal(streamID, []byte{1, 2, 3})
	require.EqualError(t, err, "StreamID too large")
}

func TestUnmarshal_InvalidFirstTwoBytes(t *testing.T) {
	_, _, err := Unmarshal([]byte{0, 0, 0, 3, 's', 'i', 'd', 1, 2, 3})
	require.EqualError(t, err, "First two bytes should be 'PC'")
}

func TestUnmarshal_InvalidHeader(t *testing.T) {
	_, _, err := Unmarshal([]byte{0, 0, 0})
	require.EqualError(t, err, "Header is too short")
}

func TestUnmarshal_LengthMismatch(t *testing.T) {
	_, _, err := Unmarshal([]byte{'P', 'C', 0, 3, 'i', 'd'})
	require.EqualError(t, err, "StreamID length mismatch")
}

func TestUnmarshal_OK(t *testing.T) {
	streamID, data, err := Unmarshal([]byte{'P', 'C', 0, 2, 'i', 'd', 'd', 'a', 't', 'a'})
	require.NoError(t, err)

	assert.Equal(t, "id", streamID)
	assert.Equal(t, "data", string(data))
}
