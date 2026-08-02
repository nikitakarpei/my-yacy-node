package yacymodel

import (
	"cmp"
	"slices"
)

func DistanceToPosition(hash Hash, position DHTPosition) DHTPosition {
	return Distance(position, RingPosition(hash))
}

func CloserToPosition(peer, than Hash, position DHTPosition) bool {
	return DistanceToPosition(peer, position) < DistanceToPosition(than, position)
}

func SeedsClosestToPosition(seeds []Seed, position DHTPosition, want int) []Seed {
	if want <= 0 {
		return nil
	}

	type rankedSeed struct {
		seed     Seed
		distance DHTPosition
	}

	ranked := make([]rankedSeed, 0, len(seeds))
	for _, seed := range seeds {
		ranked = append(
			ranked,
			rankedSeed{seed: seed, distance: DistanceToPosition(seed.Hash, position)},
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

func SeedsCloserThanPeer(seeds []Seed, than Hash, position DHTPosition) []Seed {
	closer := make([]Seed, 0, len(seeds))
	for _, seed := range seeds {
		if CloserToPosition(seed.Hash, than, position) {
			closer = append(closer, seed)
		}
	}

	return closer
}
