package vault

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"
)

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

type orderedBytes struct {
	encoded  []byte
	inverted bool
}

func (stored orderedBytes) byteAt(position int) byte {
	if stored.inverted {
		return ^stored.encoded[position]
	}

	return stored.encoded[position]
}

func (stored orderedBytes) after(length int) orderedBytes {
	return orderedBytes{encoded: stored.encoded[length:], inverted: stored.inverted}
}

func (stored orderedBytes) flipped() orderedBytes {
	return orderedBytes{encoded: stored.encoded, inverted: !stored.inverted}
}

func (stored orderedBytes) leftoverError() error {
	if len(stored.encoded) == 0 {
		return nil
	}

	return fmt.Errorf("%w: %d trailing bytes", errKeyLayoutMismatch, len(stored.encoded))
}

func parseOrderedText(stored orderedBytes) (text []byte, rest orderedBytes, err error) {
	textEndAt, escapes, err := textEndOf(stored)
	if err != nil {
		return nil, orderedBytes{}, err
	}
	rest = stored.after(textEndAt + 2)
	if escapes == 0 && !stored.inverted {
		return stored.encoded[:textEndAt], rest, nil
	}

	return unescapedTextOf(stored, textEndAt-escapes), rest, nil
}

func textEndOf(stored orderedBytes) (textEndAt, escapes int, err error) {
	for position := 0; position+1 < len(stored.encoded); position++ {
		current, next := stored.byteAt(position), stored.byteAt(position+1)
		switch {
		case current == zeroByte && next == textEnd:
			return position, escapes, nil
		case current == zeroByte && next == fullByte, current == fullByte && next == zeroByte:
			escapes++
			position++
		case current == zeroByte, current == fullByte:
			return 0, 0, fmt.Errorf("%w: bad escape in text", errKeyLayoutMismatch)
		}
	}

	return 0, 0, fmt.Errorf("%w: unterminated text", errKeyLayoutMismatch)
}

func unescapedTextOf(stored orderedBytes, textLength int) []byte {
	text := make([]byte, 0, textLength)
	for position := 0; len(text) < textLength; position++ {
		current := stored.byteAt(position)
		text = append(text, current)
		if current == zeroByte || current == fullByte {
			position++
		}
	}

	return text
}

func parseOrderedInteger(stored orderedBytes) (number int64, rest orderedBytes, err error) {
	if len(stored.encoded) == 0 {
		return 0, orderedBytes{}, fmt.Errorf("%w: missing integer", errKeyLayoutMismatch)
	}
	negative := stored.byteAt(0)&0x80 == 0
	if negative {
		stored = stored.flipped()
	}

	length := bits.LeadingZeros8(^stored.byteAt(0))
	if length == 8 && len(stored.encoded) > 1 {
		length += bits.LeadingZeros8(^stored.byteAt(1))
	}
	if length > integerBytesMax || length > len(stored.encoded) {
		return 0, orderedBytes{}, fmt.Errorf("%w: bad integer", errKeyLayoutMismatch)
	}

	var digits [integerBytesMax]byte
	integerBytes := digits[integerBytesMax-length:]
	for position := range integerBytes {
		integerBytes[position] = stored.byteAt(position)
	}
	for bit := range length + 1 {
		integerBytes[bit/8] &^= 0x80 >> (bit % 8)
	}
	magnitude := binary.BigEndian.Uint64(digits[integerBytesMax-8:])
	if digits[0]|digits[1] != 0 || magnitude > math.MaxInt64 {
		return 0, orderedBytes{}, fmt.Errorf("%w: integer out of range", errKeyLayoutMismatch)
	}

	number = int64(magnitude)
	rest = stored.after(length)
	if negative {
		return ^number, rest.flipped(), nil
	}

	return number, rest, nil
}
