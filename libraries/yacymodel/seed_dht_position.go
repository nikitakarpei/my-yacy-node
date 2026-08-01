package yacymodel

import (
	"cmp"
	"slices"
)

func DistanceToPosition(hash Hash, position DHTPosition) DHTPosition {
	return Distance(position, RingPosition(hash))
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
