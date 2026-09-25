package urlmetadataaskceilings_test

import (
	"context"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/urlmetadataaskceilings"
)

const (
	mostDocuments  = 1000
	leastDocuments = 25
	targetTime     = time.Second
	slowPeer       = "http://slow.example"
	otherPeer      = "http://other.example"
)

type setCeiling struct {
	address            string
	documentsPerSecond int
	ceiling            int
}

type handedOutCeiling struct {
	address string
	ceiling int
}

type recordedCeilings struct {
	set       []setCeiling
	handedOut []handedOutCeiling
}

func (recorded *recordedCeilings) AskCeilingSet(
	_ context.Context,
	address string,
	documentsPerSecond float64,
	ceiling int,
) {
	recorded.set = append(
		recorded.set, setCeiling{address, int(math.Round(documentsPerSecond)), ceiling},
	)
}

func (recorded *recordedCeilings) AskCeilingHandedOut(
	_ context.Context,
	address string,
	ceiling int,
) {
	recorded.handedOut = append(recorded.handedOut, handedOutCeiling{address, ceiling})
}

func ceilingsOfNoPeer() *urlmetadataaskceilings.AskCeilings {
	return urlmetadataaskceilings.New(
		mostDocuments, leastDocuments, targetTime, urlmetadataaskceilings.AskCeilingObservers{},
	)
}

func ceilingOf(t *testing.T, ceilings *urlmetadataaskceilings.AskCeilings, address string) int {
	t.Helper()

	ceiling, _ := ceilings.CeilingOf(t.Context(), address)

	return ceiling
}

func TestAnAddressWithNoSampleTakesTheMostDocuments(t *testing.T) {
	t.Parallel()

	ceiling, underTheMost := ceilingsOfNoPeer().CeilingOf(t.Context(), slowPeer)

	if ceiling != mostDocuments || underTheMost {
		t.Fatalf("CeilingOf = %d, %t; want %d, not under the most",
			ceiling, underTheMost, mostDocuments)
	}
}

func TestAnAnsweredAskSizesTheCeilingToTheDocumentsAnsweredWithinTheTargetTime(t *testing.T) {
	t.Parallel()

	ceilings := ceilingsOfNoPeer()
	ceilings.Asked(slowPeer, 1000)
	ceilings.AskAnswered(t.Context(), slowPeer, 2500*time.Millisecond)
	ceiling, underTheMost := ceilings.CeilingOf(t.Context(), slowPeer)

	if ceiling != 400 || !underTheMost || ceilingOf(t, ceilings, otherPeer) != mostDocuments {
		t.Fatalf("CeilingOf = %d, %t; want 400, under the most, and the other address at %d",
			ceiling, underTheMost, mostDocuments)
	}
}

func TestACancelledAskOfAFreshAddressSetsTheRateItProvedSoFar(t *testing.T) {
	t.Parallel()

	ceilings := ceilingsOfNoPeer()
	ceilings.Asked(slowPeer, 1000)
	ceilings.AskCancelled(t.Context(), slowPeer, 1200*time.Millisecond)

	if ceiling := ceilingOf(t, ceilings, slowPeer); ceiling != 833 {
		t.Fatalf("CeilingOf = %d, want 833", ceiling)
	}
}

func TestACancelledAskThatProvedMoreThanTheAverageLeavesIt(t *testing.T) {
	t.Parallel()

	ceilings := ceilingsOfNoPeer()
	ceilings.Asked(slowPeer, 1000)
	ceilings.AskAnswered(t.Context(), slowPeer, 2500*time.Millisecond)
	ceilings.Asked(slowPeer, 400)
	ceilings.AskCancelled(t.Context(), slowPeer, 10*time.Millisecond)

	if ceiling := ceilingOf(t, ceilings, slowPeer); ceiling != 400 {
		t.Fatalf("CeilingOf = %d, want 400", ceiling)
	}
}

func TestAFailedAskBlendsNoDocumentsIntoTheAverage(t *testing.T) {
	t.Parallel()

	ceilings := ceilingsOfNoPeer()
	ceilings.Asked(slowPeer, 1000)
	ceilings.AskAnswered(t.Context(), slowPeer, 2500*time.Millisecond)
	ceilings.Asked(slowPeer, 400)
	ceilings.AskFailed(t.Context(), slowPeer)

	if ceiling := ceilingOf(t, ceilings, slowPeer); ceiling != 200 {
		t.Fatalf("CeilingOf = %d, want 200", ceiling)
	}
}

func TestTheFirstSampleSetsTheAverageAndEachLaterSampleWeighsHalf(t *testing.T) {
	t.Parallel()

	ceilings := ceilingsOfNoPeer()
	ceilings.Asked(slowPeer, 1000)
	ceilings.AskAnswered(t.Context(), slowPeer, time.Second)
	afterTheFirstSample := ceilingOf(t, ceilings, slowPeer)
	ceilings.Asked(slowPeer, 1000)
	ceilings.AskAnswered(t.Context(), slowPeer, 3*time.Second)

	if afterTheFirstSample != 1000 || ceilingOf(t, ceilings, slowPeer) != 667 {
		t.Fatalf("ceilings = %d then %d, want 1000 then (1000+333)/2 = 667",
			afterTheFirstSample, ceilingOf(t, ceilings, slowPeer))
	}
}

func TestAnAnswerToAnAskAtTheMostThatTheCeilingDidNotLimitLeavesTheMost(t *testing.T) {
	t.Parallel()

	ceilings := ceilingsOfNoPeer()
	ceilingOf(t, ceilings, slowPeer)
	ceilings.Asked(slowPeer, 3)
	ceilings.AskAnswered(t.Context(), slowPeer, 500*time.Millisecond)

	if ceiling, underTheMost := ceilings.CeilingOf(
		t.Context(),
		slowPeer,
	); ceiling != mostDocuments ||
		underTheMost {
		t.Fatalf("CeilingOf = %d, %t; want %d, not under the most",
			ceiling, underTheMost, mostDocuments)
	}
}

func ceilingsWithSlowPeerAt400(t *testing.T) *urlmetadataaskceilings.AskCeilings {
	t.Helper()

	ceilings := ceilingsOfNoPeer()
	ceilingOf(t, ceilings, slowPeer)
	ceilings.Asked(slowPeer, 1000)
	ceilings.AskAnswered(t.Context(), slowPeer, 2500*time.Millisecond)

	return ceilings
}

func TestAnAnswerToAnAskUnderTheCeilingLeavesTheCeiling(t *testing.T) {
	t.Parallel()

	ceilings := ceilingsWithSlowPeerAt400(t)
	ceilingOf(t, ceilings, slowPeer)
	ceilings.Asked(slowPeer, 100)
	ceilings.AskAnswered(t.Context(), slowPeer, 2*time.Second)

	if ceiling := ceilingOf(t, ceilings, slowPeer); ceiling != 400 {
		t.Fatalf("CeilingOf = %d, want 400", ceiling)
	}
}

func TestAnAnswerToAnAskTheCeilingLimitedBlendsItsRateIntoTheAverage(t *testing.T) {
	t.Parallel()

	ceilings := ceilingsWithSlowPeerAt400(t)
	ceilingOf(t, ceilings, slowPeer)
	ceilings.Asked(slowPeer, 400)
	ceilings.AskAnswered(t.Context(), slowPeer, 500*time.Millisecond)

	if ceiling := ceilingOf(t, ceilings, slowPeer); ceiling != 600 {
		t.Fatalf("CeilingOf = %d, want (400+800)/2 = 600", ceiling)
	}
}

func TestACancelledAskUnderTheCeilingStillLowersTheAverage(t *testing.T) {
	t.Parallel()

	ceilings := ceilingsWithSlowPeerAt400(t)
	ceilingOf(t, ceilings, slowPeer)
	ceilings.Asked(slowPeer, 100)
	ceilings.AskCancelled(t.Context(), slowPeer, time.Second)

	if ceiling := ceilingOf(t, ceilings, slowPeer); ceiling != 250 {
		t.Fatalf("CeilingOf = %d, want (400+100)/2 = 250", ceiling)
	}
}

func TestTheCeilingStaysBetweenTheLeastAndTheMost(t *testing.T) {
	t.Parallel()

	ceilings := ceilingsOfNoPeer()
	ceilings.Asked(slowPeer, 1000)
	ceilings.AskFailed(t.Context(), slowPeer)
	ceilings.Asked(otherPeer, 1000)
	ceilings.AskAnswered(t.Context(), otherPeer, 100*time.Millisecond)

	if ceilingOf(t, ceilings, slowPeer) != leastDocuments ||
		ceilingOf(t, ceilings, otherPeer) != mostDocuments {
		t.Fatalf("ceilings = %d and %d, want %d and %d",
			ceilingOf(t, ceilings, slowPeer), ceilingOf(t, ceilings, otherPeer),
			leastDocuments, mostDocuments)
	}
}

func TestAnOutcomeWithoutAnAskChangesNothing(t *testing.T) {
	t.Parallel()

	ceilings := ceilingsOfNoPeer()
	ceilings.Asked(slowPeer, 1000)
	ceilings.AskAnswered(t.Context(), slowPeer, 2500*time.Millisecond)
	ceilings.AskFailed(t.Context(), slowPeer)

	if ceiling := ceilingOf(t, ceilings, slowPeer); ceiling != 400 {
		t.Fatalf("CeilingOf = %d, want 400", ceiling)
	}
}

func TestTheObserverSeesEachCeilingSetUnderTheMostAndEachCeilingHandedOut(t *testing.T) {
	t.Parallel()

	recorded := &recordedCeilings{}
	ceilings := urlmetadataaskceilings.New(mostDocuments, leastDocuments, targetTime, recorded)
	ceilings.Asked(otherPeer, 1000)
	ceilings.AskAnswered(t.Context(), otherPeer, time.Second)
	ceilings.Asked(slowPeer, 1000)
	ceilings.AskAnswered(t.Context(), slowPeer, 2500*time.Millisecond)
	ceilings.CeilingOf(t.Context(), slowPeer)
	ceilings.CeilingOf(t.Context(), otherPeer)

	wantSet := []setCeiling{{slowPeer, 400, 400}}
	wantHandedOut := []handedOutCeiling{{slowPeer, 400}, {otherPeer, mostDocuments}}
	if !slices.Equal(recorded.set, wantSet) || !slices.Equal(recorded.handedOut, wantHandedOut) {
		t.Fatalf("observed set %v and handed out %v, want %v and %v",
			recorded.set, recorded.handedOut, wantSet, wantHandedOut)
	}
}
