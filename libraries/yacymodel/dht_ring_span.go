package yacymodel

// DHTRingSpan is a run of ring positions from First through Last, both included,
// that does not cross the end of the ring.
type DHTRingSpan struct {
	First DHTRingPosition
	Last  DHTRingPosition
}

// Contains tells whether a position sits in the span.
func (s DHTRingSpan) Contains(position DHTRingPosition) bool {
	return position >= s.First && position <= s.Last
}
