package yacymodel

import (
	"cmp"
	"iter"
	"slices"
)

type SeedRing struct {
	seeds     []Seed
	positions []DHTRingPosition
}

func SeedRingFrom(seeds []Seed) SeedRing {
	ordered := slices.Clone(seeds)
	slices.SortStableFunc(ordered, func(seed, otherSeed Seed) int {
		return cmp.Compare(DHTRingPositionOf(seed.Hash), DHTRingPositionOf(otherSeed.Hash))
	})

	positions := make([]DHTRingPosition, len(ordered))
	for index, seed := range ordered {
		positions[index] = DHTRingPositionOf(seed.Hash)
	}

	return SeedRing{seeds: ordered, positions: positions}
}

func (r SeedRing) Seeds() []Seed { return r.seeds }

func (r SeedRing) SeedsClosestTo(position DHTRingPosition, want int) []Seed {
	closest := make([]Seed, 0, min(max(want, 0), len(r.seeds)))
	for seed := range r.SeedsClockwiseFrom(position) {
		if len(closest) == cap(closest) {
			break
		}
		closest = append(closest, seed)
	}

	return closest
}

func (r SeedRing) SeedsClockwiseFrom(position DHTRingPosition) iter.Seq[Seed] {
	return func(yield func(Seed) bool) {
		first, _ := slices.BinarySearch(r.positions, position)
		for step := range len(r.seeds) {
			if !yield(r.seeds[(first+step)%len(r.seeds)]) {
				return
			}
		}
	}
}
