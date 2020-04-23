package server

import (
	"fmt"
	"math/big"
)

const AlphabetBase16 = "1234567890abcdef"
const AlphabetBase62 = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const AlphabetBase64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

var zero big.Int

type BaseNEncoder struct {
	alphabet string
}

func NewBaseNEncoder(alphabet string) *BaseNEncoder {
	return &BaseNEncoder{alphabet}
}

func (e *BaseNEncoder) Encode(data []byte) string {
	var value big.Int
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

type BaseNDecoder struct {
	alphabet    string
	runeToValue map[rune]int
}

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

func (d *BaseNDecoder) Decode(data string) ([]byte, error) {
	var n big.Int
	n.SetInt64(int64(len(d.alphabet)))

	var factor big.Int
	var currentValue big.Int
	var value big.Int

	for i, r := range data {
		val, ok := d.runeToValue[r]
		if !ok {
			return nil, fmt.Errorf("Character %s not found in alphabet: %s", string(r), d.alphabet)
		}

		runeValue := int64(val)
		currentValue.SetInt64(runeValue)
		factor.SetInt64(int64(i)).Exp(&n, &factor, &zero)
		currentValue.SetInt64(runeValue).Mul(&currentValue, &factor)
		value.Add(&value, &currentValue)
	}

	return value.Bytes(), nil
}
