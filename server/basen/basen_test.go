package basen

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getCases(start, end int) (cases [][]byte) {
	var (
		value  big.Int
		offset big.Int
	)

	value.SetInt64(int64(start))

	for i := start; i <= end; i++ {
		value.Add(&value, &offset)
		cases = append(cases, value.Bytes())

		offset.SetInt64(1)
	}

	return
}

func TestEncodeDecode_base16(t *testing.T) {
	t.Parallel()

	encoder := NewBaseNEncoder(AlphabetBase16)
	decoder := NewBaseNDecoder(AlphabetBase16)

	for _, data := range getCases(0x1, 0xFFFF) {
		result := encoder.Encode(data)
		data2, err := decoder.Decode(result)
		assert.Nil(t, err)
		assert.Equal(t, data, data2)
	}
}

func TestEncodeDecode_base62(t *testing.T) {
	t.Parallel()

	encoder := NewBaseNEncoder(AlphabetBase62)
	decoder := NewBaseNDecoder(AlphabetBase62)

	for _, data := range getCases(0x1, 0xFFFF) {
		result := encoder.Encode(data)
		data2, err := decoder.Decode(result)
		assert.Nil(t, err)
		assert.Equal(t, data, data2)
	}
}

func TestEncodeDecode_base64(t *testing.T) {
	t.Parallel()

	encoder := NewBaseNEncoder(AlphabetBase64)
	decoder := NewBaseNDecoder(AlphabetBase64)

	for _, data := range getCases(0x1, 0xFFFF) {
		result := encoder.Encode(data)
		data2, err := decoder.Decode(result)
		assert.Nil(t, err)
		assert.Equal(t, data, data2)
	}
}

func TestDecodeError_base64(t *testing.T) {
	t.Parallel()

	decoder := NewBaseNDecoder(AlphabetBase16)
	_, err := decoder.Decode("A")
	require.NotNil(t, err, "value is nil: %v", err)
	assert.Regexp(t, "not found in alphabet", err.Error())
}
