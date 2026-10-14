package yacymodel_test

import (
	"cmp"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestSeedArcsCoverTheRingWithoutGapOrOverlap(t *testing.T) {
	ring := yacymodel.SeedRingFrom(seedsAtPositions(10, 40, 70, 95))

	assertSpansTileTheRing(t, spansOfArcs(ring.Arcs(3)))
}

func TestSeedArcWindowHoldsTheSeedsClosestToItsEnd(t *testing.T) {
	seeds := seedsAtPositions(10, 40, 70, 95)
	ring := yacymodel.SeedRingFrom(seeds)

	for _, arc := range ring.Arcs(3) {
		assertSeedsInOrder(t, arc.Window, ring.SeedsClosestTo(arc.Through, 3))
		first := arc.Spans()[0].First
		assertSeedsInOrder(t, ring.SeedsClosestTo(first, 3), arc.Window)
	}
}

func TestSeedArcPastTheRingEndGivesTwoSpans(t *testing.T) {
	ring := yacymodel.SeedRingFrom(seedsAtPositions(10, 40, 70, 95))

	var wrapping, plain int
	for _, arc := range ring.Arcs(3) {
		switch len(arc.Spans()) {
		case 1:
			plain++
		case 2:
			wrapping++
		}
	}

	if wrapping != 1 || plain != 3 {
		t.Fatalf("wrapping = %d, plain = %d, want 1 and 3", wrapping, plain)
	}
}

func TestSeedArcsJoinSeedsAtTheSamePosition(t *testing.T) {
	seeds := seedsAtPositions(10, 10, 60)
	ring := yacymodel.SeedRingFrom(seeds)

	arcs := ring.Arcs(2)

	if len(arcs) != 2 {
		t.Fatalf("arcs = %d, want 2", len(arcs))
	}
	assertSpansTileTheRing(t, spansOfArcs(arcs))
}

func TestSeedArcOfOneSeedCoversTheWholeRing(t *testing.T) {
	ring := yacymodel.SeedRingFrom(seedsAtPositions(40))

	arcs := ring.Arcs(3)

	if len(arcs) != 1 {
		t.Fatalf("arcs = %d, want 1", len(arcs))
	}
	assertSpansTileTheRing(t, arcs[0].Spans())
}

func TestSeedArcOfSeedsAllAtOnePositionCoversTheWholeRing(t *testing.T) {
	ring := yacymodel.SeedRingFrom(seedsAtPositions(40, 40))

	arcs := ring.Arcs(3)

	if len(arcs) != 1 {
		t.Fatalf("arcs = %d, want 1", len(arcs))
	}
	assertSpansTileTheRing(t, arcs[0].Spans())
}

func TestSeedArcWindowLargerThanTheRingHoldsEverySeedOnce(t *testing.T) {
	seeds := seedsAtPositions(10, 40, 70)
	ring := yacymodel.SeedRingFrom(seeds)

	for _, arc := range ring.Arcs(5) {
		if len(arc.Window) != len(seeds) {
			t.Fatalf("window = %d seeds, want %d", len(arc.Window), len(seeds))
		}
	}
}

func TestSeedArcsOfAnEmptyRingAreNone(t *testing.T) {
	if arcs := yacymodel.SeedRingFrom(nil).Arcs(3); len(arcs) != 0 {
		t.Fatalf("arcs = %v, want none", arcs)
	}
}

func spansOfArcs(arcs []yacymodel.SeedArc) []yacymodel.DHTRingSpan {
	var spans []yacymodel.DHTRingSpan
	for _, arc := range arcs {
		spans = append(spans, arc.Spans()...)
	}

	return spans
}

func assertSpansTileTheRing(t *testing.T, spans []yacymodel.DHTRingSpan) {
	t.Helper()

	ordered := slices.Clone(spans)
	slices.SortFunc(ordered, func(span, otherSpan yacymodel.DHTRingSpan) int {
		return cmp.Compare(span.First, otherSpan.First)
	})
	next := yacymodel.DHTRingPosition(0)
	for _, span := range ordered {
		if span.First != next {
			t.Fatalf("span starts at %d, want %d", span.First, next)
		}
		next = span.Last + 1
	}
	if ordered[len(ordered)-1].Last != yacymodel.MaxDHTRingPosition {
		t.Fatalf("spans end at %d, want the ring end", ordered[len(ordered)-1].Last)
	}
}
