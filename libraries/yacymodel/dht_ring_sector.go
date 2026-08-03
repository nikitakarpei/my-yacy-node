package yacymodel

import "math/bits"

// DHTRingSector is one of the equal arcs the DHT ring is cut into for
// reporting where on the ring an object sits.
type DHTRingSector uint

const MaxDHTRingSector = DHTRingSector(1)<<6 - 1

// DHTRingSectorOf is the sector a DHT position falls in.
func DHTRingSectorOf(position DHTPosition) DHTRingSector {
	return DHTRingSector(position >> (63 - bits.Len(uint(MaxDHTRingSector))))
}
