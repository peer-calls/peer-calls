package basen

import (
	"math/big"

	"github.com/juju/errors"
)

const (
	AlphabetBase16 = "1234567890abcdef"
	AlphabetBase62 = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	AlphabetBase64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
)

// BaseNEncoder is a generic base-N encoder.
type BaseNEncoder struct {
	alphabet string
}

// NewBaseNEncoder creates a new instance of BaseNEncoder using the provided
// alphabet.
func NewBaseNEncoder(alphabet string) *BaseNEncoder {
	return &BaseNEncoder{alphabet}
}

// Encode encodes the binary data into base-64 encoded strings.
func (e *BaseNEncoder) Encode(data []byte) string {
	var (
		value big.Int
		zero  big.Int
	)

	value.SetBytes(data)

	baseInt64 := int64(len(e.alphabet))

	var base big.Int

	result := []byte{}

	for value.Cmp(&zero) != 0 {
		base.SetInt64(baseInt64)
		_, remainder := value.DivMod(&value, &base, &base)
		char := e.alphabet[remainder.Int64()]
		result = append(result, char)
	}

	return string(result)
}

// BaseNDecoder is a generic base-N decoder.
type BaseNDecoder struct {
	alphabet    string
	runeToValue map[rune]int
}

// NewBaseNDecoder creates a new instance of BaseNDecoder using the provided
// alphabet.
func NewBaseNDecoder(alphabet string) *BaseNDecoder {
	runeToValue := make(map[rune]int, len(alphabet))

	for i, r := range alphabet {
		runeToValue[r] = i
	}

	return &BaseNDecoder{
		alphabet:    alphabet,
		runeToValue: runeToValue,
	}
}

// Decode decodes the string base-N data into bytes and returns any error that
// might have occurred.
func (d *BaseNDecoder) Decode(data string) ([]byte, error) {
	var n big.Int

	n.SetInt64(int64(len(d.alphabet)))

	var (
		factor       big.Int
		currentValue big.Int
		value        big.Int
		zero         big.Int
	)

	for i, r := range data {
		val, ok := d.runeToValue[r]
		if !ok {
			return nil, errors.Errorf("Character %s not found in alphabet: %s", string(r), d.alphabet)
		}

		runeValue := int64(val)
		currentValue.SetInt64(runeValue)
		factor.SetInt64(int64(i)).Exp(&n, &factor, &zero)
		currentValue.SetInt64(runeValue).Mul(&currentValue, &factor)
		value.Add(&value, &currentValue)
	}

	return value.Bytes(), nil
}
