package vault

import "encoding/binary"

const (
	zeroByte        byte = 0x00
	fullByte        byte = 0xFF
	textEnd         byte = 0x01
	integerBytesMax      = 10
)

func appendOrderedText[T string | []byte](encoded []byte, text T) []byte {
	for position := range len(text) {
		switch text[position] {
		case zeroByte:
			encoded = append(encoded, zeroByte, fullByte)
		case fullByte:
			encoded = append(encoded, fullByte, zeroByte)
		default:
			encoded = append(encoded, text[position])
		}
	}

	return append(encoded, zeroByte, textEnd)
}

func appendOrderedInteger(encoded []byte, number int64) []byte {
	magnitude := uint64(number) //nolint:gosec // G115: the bits are reinterpreted on purpose.
	if number < 0 {
		magnitude = ^magnitude
	}

	length := 1
	for length*7-1 < 64 && magnitude >= 1<<(length*7-1) {
		length++
	}

	var digits [integerBytesMax]byte
	binary.BigEndian.PutUint64(digits[integerBytesMax-8:], magnitude)
	integerBytes := digits[integerBytesMax-length:]
	for bit := range length {
		integerBytes[bit/8] |= 0x80 >> (bit % 8)
	}
	if number < 0 {
		invert(integerBytes)
	}

	return append(encoded, integerBytes...)
}

func invert(encoded []byte) {
	for position := range encoded {
		encoded[position] ^= fullByte
	}
}
