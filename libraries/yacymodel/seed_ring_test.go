package yacymodel_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestSeedRingWalksClockwiseFromThePosition(t *testing.T) {
	seeds := seedsAtPositions(10, 40, 70, 95)
	ring := yacymodel.SeedRingFrom(seedsInOrder(seeds, 2, 0, 3, 1))

	walked := slices.Collect(ring.SeedsClockwiseFrom(positionOf(seeds[1]) + 1))

	assertSeedsInOrder(t, walked, seedsInOrder(seeds, 2, 3, 0, 1))
}

func TestSeedRingWalksFromTheSeedAtThePosition(t *testing.T) {
	seeds := seedsAtPositions(10, 40, 70)
	ring := yacymodel.SeedRingFrom(seeds)

	walked := slices.Collect(ring.SeedsClockwiseFrom(positionOf(seeds[1])))

	assertSeedsInOrder(t, walked, seedsInOrder(seeds, 1, 2, 0))
}

func TestSeedRingKeepsTheClosestSeedsWanted(t *testing.T) {
	seeds := seedsAtPositions(10, 40, 70)
	ring := yacymodel.SeedRingFrom(seeds)

	closest := ring.SeedsClosestTo(positionOf(seeds[2])+1, 2)

	assertSeedsInOrder(t, closest, seedsInOrder(seeds, 0, 1))
}

func TestSeedRingKeepsEverySeedWhenMoreAreWanted(t *testing.T) {
	seeds := seedsAtPositions(10, 40)
	ring := yacymodel.SeedRingFrom(seeds)

	closest := ring.SeedsClosestTo(positionOf(seeds[0]), 5)

	assertSeedsInOrder(t, closest, seeds)
}

func TestSeedRingWalksNothingWithoutSeeds(t *testing.T) {
	ring := yacymodel.SeedRingFrom(nil)

	if walked := slices.Collect(ring.SeedsClockwiseFrom(0)); len(walked) != 0 {
		t.Fatalf("walked = %v, want nothing", walked)
	}
}

func TestSeedRingHoldsEverySeed(t *testing.T) {
	seeds := seedsAtPositions(10, 40)
	ring := yacymodel.SeedRingFrom(seedsInOrder(seeds, 1, 0))

	assertSeedsInOrder(t, ring.Seeds(), seeds)
}

func seedsAtPositions(percents ...uint64) []yacymodel.Seed {
	seeds := make([]yacymodel.Seed, 0, len(percents))
	for _, percent := range percents {
		seeds = append(seeds, seedAt(hashNear(percent)))
	}

	return seeds
}

func hashNear(percent uint64) yacymodel.Hash {
	want := yacymodel.MaxDHTRingPosition / 100 * yacymodel.DHTRingPosition(percent)
	for attempt := 0; ; attempt++ {
		hash := yacymodel.WordHash("seed" + strconv.Itoa(attempt))
		if yacymodel.DHTRingPositionOf(hash)-want < yacymodel.MaxDHTRingPosition/100 {
			return hash
		}
	}
}

func positionOf(seed yacymodel.Seed) yacymodel.DHTRingPosition {
	return yacymodel.DHTRingPositionOf(seed.Hash)
}

func seedsInOrder(seeds []yacymodel.Seed, indexes ...int) []yacymodel.Seed {
	ordered := make([]yacymodel.Seed, 0, len(indexes))
	for _, index := range indexes {
		ordered = append(ordered, seeds[index])
	}

	return ordered
}

func assertSeedsInOrder(t *testing.T, seeds, want []yacymodel.Seed) {
	t.Helper()

	if !slices.EqualFunc(seeds, want, func(seed, wantSeed yacymodel.Seed) bool {
		return seed.Hash == wantSeed.Hash
	}) {
		t.Fatalf("seeds = %v, want %v", seeds, want)
	}
}
