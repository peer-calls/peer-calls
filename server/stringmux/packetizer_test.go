package stringmux

import (
	"testing"

	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshal(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	streamID := string(make([]byte, 0xFF+1))
	_, err := Marshal(streamID, []byte{1, 2, 3})
	require.Equal(t, errors.Cause(err), ErrStreamIDTooLarge)
}

func TestUnmarshal_InvalidFirstTwoBytes(t *testing.T) {
	t.Parallel()

	_, _, err := Unmarshal([]byte{123, 2, 'H', 'I', 1, 1})
	require.Equal(t, errors.Cause(err), ErrInvalidHeader)
}

func TestUnmarshal_InvalidHeader(t *testing.T) {
	t.Parallel()

	_, _, err := Unmarshal([]byte{StringMuxByte})
	require.Equal(t, errors.Cause(err), ErrInvalidHeader)
}

func TestUnmarshal_LengthMismatch(t *testing.T) {
	t.Parallel()

	_, _, err := Unmarshal([]byte{StringMuxByte, 3, 'T'})
	require.Equal(t, errors.Cause(err), ErrInvalidHeader)
}

func TestUnmarshal_OK(t *testing.T) {
	t.Parallel()

	streamID, data, err := Unmarshal([]byte{StringMuxByte, 2, 'i', 'd', 'd', 'a', 't', 'a'})
	require.NoError(t, err)

	assert.Equal(t, "id", streamID)
	assert.Equal(t, "data", string(data))
}
