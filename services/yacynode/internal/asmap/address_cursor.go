package asmap

import "math/bits"

type addressCursor struct {
	address  [16]byte
	position int
}

func (c *addressCursor) readBit() bool {
	set := c.address[c.position/8]>>(7-c.position%8)&1 == 1
	c.position++

	return set
}

func (c *addressCursor) readMatching(match uint32) bool {
	for position := bits.Len32(match) - 2; position >= 0; position-- {
		if c.readBit() != (match>>position&1 == 1) {
			return false
		}
	}

	return true
}
