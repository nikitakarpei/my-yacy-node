package postingstree

import (
	"bytes"
	"errors"
	"fmt"
)

const (
	KeyByteLength = 9
	Depth         = KeyByteLength * 8
)

var errBadPosition = errors.New("bad postings tree position")

type Position struct {
	prefix [KeyByteLength]byte
	depth  uint8
}

func LeafAt(key [KeyByteLength]byte) Position {
	return Position{prefix: key, depth: Depth}
}

func (p Position) childAt(bit byte) Position {
	child := p
	child.prefix[p.depth/8] |= bit << (7 - p.depth%8)
	child.depth++

	return child
}

func (p Position) height() uint8 {
	return Depth - p.depth
}

func (p Position) bitAt(index uint8) byte {
	return p.prefix[index/8] >> (7 - index%8) & 1
}

func (p Position) compare(other Position) int {
	if order := bytes.Compare(p.prefix[:], other.prefix[:]); order != 0 {
		return order
	}

	return int(p.depth) - int(other.depth)
}

func (p Position) appendedTo(encoding []byte) []byte {
	encoding = append(encoding, p.depth)

	return append(encoding, p.prefix[:prefixLengthOf(p.depth)]...)
}

func prefixLengthOf(depth uint8) int {
	return (int(depth) + 7) / 8
}

func positionFrom(raw []byte) (position Position, rest []byte, err error) {
	depth := raw[0]
	if depth > Depth {
		return position, nil, fmt.Errorf("%w: depth %d", errBadPosition, depth)
	}
	prefixLength := prefixLengthOf(depth)
	if len(raw) < 1+prefixLength {
		return position, nil, fmt.Errorf("%w: truncated prefix", errBadPosition)
	}
	position.depth = depth
	copy(position.prefix[:], raw[1:1+prefixLength])
	if position != position.truncatedTo(depth) {
		return position, nil, fmt.Errorf("%w: bits past depth %d", errBadPosition, depth)
	}

	return position, raw[1+prefixLength:], nil
}

func (p Position) truncatedTo(depth uint8) Position {
	truncated := Position{depth: depth}
	wholeBytes := depth / 8
	copy(truncated.prefix[:wholeBytes], p.prefix[:wholeBytes])
	if partialBits := depth % 8; partialBits > 0 {
		truncated.prefix[wholeBytes] = p.prefix[wholeBytes] & (0xFF << (8 - partialBits))
	}

	return truncated
}

func (p Position) sibling() (Position, bool) {
	if p.depth == 0 {
		return p, false
	}
	sibling := p
	lastBit := p.depth - 1
	sibling.prefix[lastBit/8] ^= 1 << (7 - lastBit%8)

	return sibling, true
}

func (p Position) covers(other Position) bool {
	return other.depth >= p.depth && other.truncatedTo(p.depth) == p
}
