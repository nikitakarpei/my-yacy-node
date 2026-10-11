package yacymodel

func CloserToDHTRingPosition(peer, than Hash, position DHTRingPosition) bool {
	return position.DistanceTo(DHTRingPositionOf(peer)) <
		position.DistanceTo(DHTRingPositionOf(than))
}

func SeedsPerDHTRingSector(seeds []Seed) []int {
	perSector := make([]int, DHTRingSectorCount)
	for _, seed := range seeds {
		perSector[DHTRingSectorOf(DHTRingPositionOf(seed.Hash))]++
	}

	return perSector
}
