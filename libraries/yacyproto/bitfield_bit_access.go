package yacyproto

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var errInvalidBitfield = errors.New("invalid bitfield")

type bitfield []byte

func decodeBitfield(encoded string) (bitfield, error) {
	raw, err := yacymodel.Decode(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidBitfield, err)
	}
	return bitfield(raw), nil
}

func (b bitfield) get(pos int) bool {
	slot := pos >> 3
	if pos < 0 || slot >= len(b) {
		return false
	}
	return b[slot]&(1<<(pos%8)) != 0
}

func (b bitfield) allSet(bits int) bool {
	for pos := range bits {
		if !b.get(pos) {
			return false
		}
	}
	return true
}

func (b bitfield) setBit(pos int, value bool) {
	slot := pos >> 3
	if pos < 0 || slot >= len(b) {
		return
	}
	if value {
		b[slot] |= 1 << (pos % 8)
	} else {
		b[slot] &^= 1 << (pos % 8)
	}
}
