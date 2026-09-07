package yacymodel

import (
	"fmt"
	"math/bits"
)

// DHTRingPosition is where something sits on the DHT ring. The ring is 63 bits
// wide and ends at MaxDHTRingPosition.
type DHTRingPosition uint64

const MaxDHTRingPosition = DHTRingPosition(1)<<63 - 1

// DHTRingDistance is how far apart two positions are. You always measure it
// forward around the ring.
type DHTRingDistance uint64

// DHTRingPartitions is how many equal parts the ring is cut into. The count is
// always a power of two, so you build it from the exponent and not from the
// count.
type DHTRingPartitions uint

// DHTRingPartitionsFromExponent makes 1<<exponent partitions. The exponent is
// how many of the high bits of a posting's position come from its url hash
// (Distribution.java:49-52).
func DHTRingPartitionsFromExponent(exponent uint) (DHTRingPartitions, error) {
	if exponent >= 63 {
		return 0, fmt.Errorf("dht ring partition exponent %d out of range [0,63)", exponent)
	}
	return DHTRingPartitions(1) << exponent, nil
}

func (p DHTRingPartitions) shiftLength() uint {
	return 63 - uint(bits.Len(uint(p))-1)
}

const dhtRingPositionSymbols = 10

// DHTRingPositionOf puts a hash on the ring in base64 order. Words and peers
// go on the same ring, so you can compare them (Distribution.java:74-78,
// horizontalDHTPosition).
func DHTRingPositionOf(hash Hash) DHTRingPosition {
	symbols := hash.String()
	read := min(len(symbols), dhtRingPositionSymbols)

	var position uint64
	for i := range read {
		position = position<<6 | uint64(decodeTable[symbols[i]]&0x3f)
	}
	for range dhtRingPositionSymbols - read {
		position <<= 6
	}

	return DHTRingPosition(position<<3 | 7)
}

// HashFromDHTRingPosition turns a position back into a hash that sits there. It
// reverses DHTRingPositionOf, but it can only recover the first ten symbols
// (Distribution.java:111-116, positionToHash).
func HashFromDHTRingPosition(position DHTRingPosition) Hash {
	remaining := uint64(position) >> 3
	symbols := make([]byte, HashLength)
	for i := dhtRingPositionSymbols - 1; i >= 0; i-- {
		symbols[i] = Alphabet[remaining&0x3f]
		remaining >>= 6
	}
	for i := dhtRingPositionSymbols; i < HashLength; i++ {
		symbols[i] = Alphabet[len(Alphabet)-1]
	}

	hash, _ := ParseHash(string(symbols))

	return hash
}

// DistanceTo is how far you go forward around the ring to reach another
// position. The ring wraps at MaxDHTRingPosition (Distribution.java:101-103,
// horizontalDHTDistance).
func (p DHTRingPosition) DistanceTo(other DHTRingPosition) DHTRingDistance {
	if other >= p {
		return DHTRingDistance(other - p)
	}

	return DHTRingDistance((MaxDHTRingPosition - p) + other + 1)
}

// FractionOfDHTRing is this distance as a part of the whole ring, from 0 to 1.
func (d DHTRingDistance) FractionOfDHTRing() float64 {
	return float64(d) / float64(uint64(MaxDHTRingPosition)+1)
}

// DHTRingPositionOfPosting puts one posting on the ring. The low bits come from
// the word hash and the high bits from the url hash. This spreads the postings
// of one word over the partitions, so they do not all go to the peer nearest
// that word (Distribution.java:130-133, verticalDHTPosition).
func DHTRingPositionOfPosting(
	posting RWIPosting,
	partitions DHTRingPartitions,
) DHTRingPosition {
	wordPosition := DHTRingPositionOf(posting.WordHash)
	urlPosition := DHTRingPositionOf(posting.URLHash.hash)
	mask := DHTRingPosition(uint64(1)<<partitions.shiftLength() - 1)

	return wordPosition&mask | urlPosition&^mask
}

// DistanceFromPostingsOfWord is how far this position is past the postings of a
// word. In each partition there is one position where the postings of a word go.
// This position is inside one partition. The distance goes forward from the
// posting position in that partition to this position. Zero means that the
// postings of the word go here.
func (p DHTRingPosition) DistanceFromPostingsOfWord(
	word Hash,
	partitions DHTRingPartitions,
) DHTRingDistance {
	wordPosition := DHTRingPositionOf(word)
	mask := uint64(1)<<partitions.shiftLength() - 1

	return DHTRingDistance((uint64(p) - uint64(wordPosition)) & mask)
}

func DHTRingPositionOfWordInPartition(
	word Hash,
	partition uint,
	partitions DHTRingPartitions,
) DHTRingPosition {
	shift := partitions.shiftLength()
	mask := DHTRingPosition(uint64(1)<<shift - 1)

	return DHTRingPositionOf(word)&mask | DHTRingPosition(uint64(partition)<<shift)&^mask
}
