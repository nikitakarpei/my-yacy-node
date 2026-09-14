package yacymodel

import (
	"errors"
	"fmt"
)

const HostHashLength = 6

var ErrInvalidHostHash = errors.New("invalid host hash")

type HostHash struct{ value string }

func ParseHostHash(s string) (HostHash, error) {
	if len(s) != HostHashLength {
		return HostHash{}, fmt.Errorf(
			"%w: length %d, want %d",
			ErrInvalidHostHash,
			len(s),
			HostHashLength,
		)
	}
	for i := range len(s) {
		if decodeTable[s[i]] < 0 {
			return HostHash{}, fmt.Errorf("%w: %q", ErrInvalidHostHash, s[i])
		}
	}
	return HostHash{value: s}, nil
}

func (h HostHash) String() string { return h.value }
