package yacymodel

import (
	"bytes"
	"errors"
	"fmt"
)

const postingsTreeDepth = HashByteLength * 8

var errBadPostingsTreePosition = errors.New("bad postings tree position")

type postingsTreePosition struct {
	prefix [HashByteLength]byte
	depth  uint8
}

func (p postingsTreePosition) height() uint8 {
	return postingsTreeDepth - p.depth
}

func (p postingsTreePosition) bitAt(index uint8) byte {
	return bitOf(p.prefix[:], uint(index))
}

func bitOf(raw []byte, index uint) byte {
	return raw[index/8] >> (7 - index%8) & 1
}

func (p postingsTreePosition) childAt(bit byte) postingsTreePosition {
	child := p
	child.prefix[p.depth/8] |= bit << (7 - p.depth%8)
	child.depth++

	return child
}

func (p postingsTreePosition) compare(other postingsTreePosition) int {
	if order := bytes.Compare(p.prefix[:], other.prefix[:]); order != 0 {
		return order
	}

	return int(p.depth) - int(other.depth)
}

func (p postingsTreePosition) appendedTo(encoding []byte) []byte {
	encoding = append(encoding, p.depth)

	return append(encoding, p.prefix[:prefixLengthOf(p.depth)]...)
}

func prefixLengthOf(depth uint8) int {
	return (int(depth) + 7) / 8
}

func positionFrom(raw []byte) (position postingsTreePosition, rest []byte, err error) {
	depth := raw[0]
	if depth > postingsTreeDepth {
		return position, nil, fmt.Errorf("%w: depth %d", errBadPostingsTreePosition, depth)
	}
	prefixLength := prefixLengthOf(depth)
	if len(raw) < 1+prefixLength {
		return position, nil, fmt.Errorf("%w: truncated prefix", errBadPostingsTreePosition)
	}
	position.depth = depth
	copy(position.prefix[:], raw[1:1+prefixLength])
	if position != position.truncatedTo(depth) {
		return position, nil, fmt.Errorf(
			"%w: bits past depth %d",
			errBadPostingsTreePosition,
			depth,
		)
	}

	return position, raw[1+prefixLength:], nil
}

func (p postingsTreePosition) truncatedTo(depth uint8) postingsTreePosition {
	truncated := postingsTreePosition{depth: depth}
	wholeBytes := depth / 8
	copy(truncated.prefix[:wholeBytes], p.prefix[:wholeBytes])
	if partialBits := depth % 8; partialBits > 0 {
		truncated.prefix[wholeBytes] = p.prefix[wholeBytes] & (0xFF << (8 - partialBits))
	}

	return truncated
}

func (p postingsTreePosition) sibling() (postingsTreePosition, bool) {
	if p.depth == 0 {
		return p, false
	}
	sibling := p
	lastBit := p.depth - 1
	sibling.prefix[lastBit/8] ^= 1 << (7 - lastBit%8)

	return sibling, true
}

func (p postingsTreePosition) covers(other postingsTreePosition) bool {
	return other.depth >= p.depth && other.truncatedTo(p.depth) == p
}
