package yacymodel

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"strings"
)

const HashLength = 12

const HashByteLength = HashLength * 6 / 8

const ReservedPrefix = "_____"

var ErrInvalidHash = errors.New("invalid hash")

type Hash struct {
	raw     [HashByteLength]byte
	present bool
}

func ParseHash(s string) (Hash, error) {
	if len(s) != HashLength {
		return Hash{}, fmt.Errorf("%w: length %d, want %d", ErrInvalidHash, len(s), HashLength)
	}
	for i := range len(s) {
		if decodeTable[s[i]] < 0 {
			return Hash{}, fmt.Errorf("%w: %q", ErrInvalidHash, s[i])
		}
	}

	return hashOfSymbols(s), nil
}

func hashOfSymbols[T string | []byte](symbols T) Hash {
	hash := Hash{present: true}
	var buffer uint32
	var bits uint
	filled := 0
	for i := range len(symbols) {
		//nolint:gosec // G115: every symbol is from Alphabet, so its value is in [0,64).
		buffer = buffer<<6 | uint32(decodeTable[symbols[i]])
		bits += 6
		if bits >= 8 {
			bits -= 8
			hash.raw[filled] = byte((buffer >> bits) & 0xff)
			filled++
		}
	}

	return hash
}

func ParseHashBytes(raw []byte) (Hash, error) {
	if len(raw) != HashByteLength {
		return Hash{}, fmt.Errorf(
			"%w: %d bytes, want %d",
			ErrInvalidHash,
			len(raw),
			HashByteLength,
		)
	}
	hash := Hash{present: true}
	copy(hash.raw[:], raw)

	return hash, nil
}

func (h Hash) Reserved() bool {
	var text [HashLength]byte

	return bytes.HasPrefix(h.appendText(text[:0]), []byte(ReservedPrefix))
}

func (h Hash) appendText(buffer []byte) []byte {
	if !h.present {
		return buffer
	}

	return appendEncoded(buffer, h.raw[:])
}

func (h Hash) IsZero() bool { return !h.present }

func (h Hash) String() string {
	var text [HashLength]byte

	return string(h.appendText(text[:0]))
}

func (h Hash) Bytes() []byte {
	return h.AppendBytes(nil)
}

func (h Hash) AppendBytes(buffer []byte) []byte {
	if !h.present {
		return buffer
	}

	return append(buffer, h.raw[:]...)
}

func (h Hash) CompareInAlphabetOrder(other Hash) int {
	var raw, otherRaw [HashByteLength]byte

	return bytes.Compare(h.AppendBytes(raw[:0]), other.AppendBytes(otherRaw[:0]))
}

func (h Hash) MarshalText() ([]byte, error) {
	return h.appendText(nil), nil
}

func (h *Hash) UnmarshalText(text []byte) error {
	parsed, err := ParseHash(string(text))
	if err != nil {
		return err
	}
	*h = parsed

	return nil
}

func WordHash(word string) Hash {
	sum := md5.Sum([]byte(strings.ToLower(word)))
	h := []byte(Encode(sum[:])[:HashLength])
	lowByte, highByte := Alphabet[0], Alphabet[len(Alphabet)-1]
	for h[0] == highByte && h[1] == highByte && h[2] == highByte &&
		h[3] == highByte && h[4] == highByte {
		copy(h, h[1:])
		h[HashLength-1] = lowByte
	}

	return hashOfSymbols(h)
}
