package yacymodel

import (
	"cmp"
	"slices"
)

func CloserToDHTRingPosition(peer, than Hash, position DHTRingPosition) bool {
	return position.DistanceTo(DHTRingPositionOf(peer)) <
		position.DistanceTo(DHTRingPositionOf(than))
}

func SeedsClosestToDHTRingPosition(
	seeds []Seed,
	position DHTRingPosition,
	want int,
) []Seed {
	if want <= 0 {
		return nil
	}

	type rankedSeed struct {
		seed     Seed
		distance DHTRingDistance
	}

	ranked := make([]rankedSeed, 0, len(seeds))
	for _, seed := range seeds {
		ranked = append(
			ranked,
			rankedSeed{
				seed:     seed,
				distance: position.DistanceTo(DHTRingPositionOf(seed.Hash)),
			},
		)
	}
	slices.SortFunc(ranked, func(a, b rankedSeed) int {
		return cmp.Compare(a.distance, b.distance)
	})
	if len(ranked) > want {
		ranked = ranked[:want]
	}

	closest := make([]Seed, len(ranked))
	for i, entry := range ranked {
		closest[i] = entry.seed
	}

	return closest
}

// SeedsPerDHTRingSector counts the seeds in every sector of the DHT ring,
// including the sectors that hold none.
func SeedsPerDHTRingSector(seeds []Seed) []int {
	perSector := make([]int, MaxDHTRingSector+1)
	for _, seed := range seeds {
		perSector[DHTRingSectorOf(DHTRingPositionOf(seed.Hash))]++
	}

	return perSector
}

func SeedsCloserToDHTRingPositionThanPeer(
	seeds []Seed,
	than Hash,
	position DHTRingPosition,
) []Seed {
	closer := make([]Seed, 0, len(seeds))
	for _, seed := range seeds {
		if CloserToDHTRingPosition(seed.Hash, than, position) {
			closer = append(closer, seed)
		}
	}

	return closer
}
