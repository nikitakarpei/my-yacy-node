package yacymodel

import (
	"fmt"
	"math/bits"
)

// DHTPosition places an object on the 63-bit DHT ring, closed at the high end.
type DHTPosition uint64

const MaxDHTPosition = DHTPosition(1)<<63 - 1

// DHTRingPartitions is how many equal partitions the DHT ring is divided
// into. It is always a power of two, so it is constructed only from its
// exponent rather than an arbitrary count.
type DHTRingPartitions uint

// DHTRingPartitionsFromExponent builds a DHTRingPartitions of 1<<exponent
// partitions. exponent is the number of high bits of a posting's position
// that its url hash contributes (Distribution.java:49-52).
func DHTRingPartitionsFromExponent(exponent uint) (DHTRingPartitions, error) {
	if exponent >= 63 {
		return 0, fmt.Errorf("dht ring partition exponent %d out of range [0,63)", exponent)
	}
	return DHTRingPartitions(1) << exponent, nil
}

func (p DHTRingPartitions) shiftLength() uint {
	return 63 - uint(bits.Len(uint(p))-1)
}

// WordPosition places a word on the DHT ring by the base64 order of its
// hash (Distribution.java:74-78, horizontalDHTPosition).
func WordPosition(word Hash) (DHTPosition, error) {
	c, err := cardinal(word.String())
	if err != nil {
		return 0, fmt.Errorf("word position: %w", err)
	}
	return DHTPosition(c), nil
}

// PostingPosition places one word's posting for one url on the DHT ring: the
// low 63-e bits come from the word hash and the high e bits from the url
// hash, so that one word's postings spread across up to 1<<e partitions
// instead of piling onto the single peer nearest that word
// (Distribution.java:130-133, verticalDHTPosition).
func PostingPosition(word Hash, url URLHash, partitions DHTRingPartitions) (DHTPosition, error) {
	wordPos, err := cardinal(word.String())
	if err != nil {
		return 0, fmt.Errorf("posting position: %w", err)
	}
	urlPos, err := cardinal(url.String())
	if err != nil {
		return 0, fmt.Errorf("posting position: %w", err)
	}
	mask := uint64(1)<<partitions.shiftLength() - 1
	return DHTPosition(wordPos&mask | urlPos&^mask), nil
}

// PositionHash reverses WordPosition, computing a hash whose position is the
// given one (Distribution.java:111-116, positionToHash).
func PositionHash(pos DHTPosition) Hash {
	c := uint64(pos) >> 3
	b := make([]byte, HashLength)
	for p := 9; p >= 0; p-- {
		b[p] = Alphabet[c&0x3f]
		c >>= 6
	}
	b[10] = Alphabet[len(Alphabet)-1]
	b[11] = Alphabet[len(Alphabet)-1]
	h, _ := ParseHash(string(b))
	return h
}

// Distance is the cardinal number of positions from one DHT position to
// another, going forward around the ring closed at MaxDHTPosition
// (Distribution.java:101-103, horizontalDHTDistance).
func Distance(from, to DHTPosition) DHTPosition {
	if to >= from {
		return to - from
	}
	return (MaxDHTPosition - from) + to + 1
}
