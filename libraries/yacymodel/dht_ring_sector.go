package yacymodel

import "math/bits"

// DHTRingSector is one of the equal arcs of the ring. Use the sectors to report
// where on the ring something sits.
type DHTRingSector uint

const MaxDHTRingSector = DHTRingSector(1)<<6 - 1

// DHTRingSectorOf is the sector that holds a position.
func DHTRingSectorOf(position DHTRingPosition) DHTRingSector {
	return DHTRingSector(position >> (63 - bits.Len(uint(MaxDHTRingSector))))
}
