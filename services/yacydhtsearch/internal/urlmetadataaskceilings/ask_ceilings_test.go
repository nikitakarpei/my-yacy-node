package urlmetadataaskceilings_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/urlmetadataaskceilings"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	mostDocuments  = 1000
	leastDocuments = 25
	targetTime     = time.Second
	peerAddress    = "http://peer.example"
)

type setCeiling struct {
	address string
	pace    time.Duration
	ceiling int
}

type recordedCeilings struct {
	setCeilings []setCeiling
}

func (recorded *recordedCeilings) AskCeilingSet(
	_ context.Context,
	address string,
	pace time.Duration,
	ceiling int,
) {
	recorded.setCeilings = append(recorded.setCeilings, setCeiling{
		address: address,
		pace:    pace,
		ceiling: ceiling,
	})
}

type peerPacesInAMap map[string]time.Duration

func (paces peerPacesInAMap) Read(
	_ context.Context,
	address string,
) yacymodel.Optional[time.Duration] {
	pace, remembered := paces[address]
	if !remembered {
		return yacymodel.None[time.Duration]()
	}

	return yacymodel.Some(pace)
}

func (paces peerPacesInAMap) Update(
	ctx context.Context,
	address string,
	updated func(pace yacymodel.Optional[time.Duration]) time.Duration,
) time.Duration {
	paces[address] = updated(paces.Read(ctx, address))

	return paces[address]
}

func askCeilings() *urlmetadataaskceilings.AskCeilings {
	return urlmetadataaskceilings.New(
		mostDocuments,
		leastDocuments,
		targetTime,
		peerPacesInAMap{},
		&recordedCeilings{},
	)
}

func askCeilingsWithAPeerAtFourHundred(t *testing.T) *urlmetadataaskceilings.AskCeilings {
	t.Helper()

	ceilings := askCeilings()
	ceilings.AskAnswered(t.Context(), peerAddress, mostDocuments, 2500*time.Millisecond)

	return ceilings
}

func TestAnUnknownAddressIsAskedTheMost(t *testing.T) {
	t.Parallel()

	if ceiling := askCeilings().CeilingOf(t.Context(), peerAddress); ceiling != mostDocuments {
		t.Fatalf("ceiling of an unknown address = %d, want %d", ceiling, mostDocuments)
	}
}

func TestAnAnsweredLimitedAskSizesTheCeilingToTheDocumentsAnsweredWithinTheTargetTime(
	t *testing.T,
) {
	t.Parallel()

	ceilings := askCeilingsWithAPeerAtFourHundred(t)

	if ceiling := ceilings.CeilingOf(t.Context(), peerAddress); ceiling != 400 {
		t.Fatalf("ceiling after 1000 documents answered in 2.5s = %d, want 400", ceiling)
	}
}

func TestAnAnswerToAnAskUnderTheCeilingChangesNothing(t *testing.T) {
	t.Parallel()

	ceilings := askCeilingsWithAPeerAtFourHundred(t)
	ceilings.AskAnswered(t.Context(), peerAddress, 200, 10*time.Second)

	if ceiling := ceilings.CeilingOf(t.Context(), peerAddress); ceiling != 400 {
		t.Fatalf("ceiling after a slow ask under the ceiling = %d, want 400", ceiling)
	}
}

func TestAnAnswerToAnAskTheCeilingDidNotLimitChangesNothing(t *testing.T) {
	t.Parallel()

	ceilings := askCeilings()
	ceilings.AskAnswered(t.Context(), peerAddress, mostDocuments-1, 10*time.Second)

	if ceiling := ceilings.CeilingOf(t.Context(), peerAddress); ceiling != mostDocuments {
		t.Fatalf("ceiling after a slow ask of fewer documents than the most = %d, want %d",
			ceiling, mostDocuments)
	}
}

func TestACancelledAskOfAFreshAddressSetsWhatItProved(t *testing.T) {
	t.Parallel()

	ceilings := askCeilings()
	ceilings.AskCancelled(t.Context(), peerAddress, mostDocuments, 2*time.Second)

	if ceiling := ceilings.CeilingOf(t.Context(), peerAddress); ceiling != 500 {
		t.Fatalf("ceiling after 1000 documents cancelled after 2s = %d, want 500", ceiling)
	}
}

func TestACancelledAskThatProvedLessTimeThanTheAverageLeavesIt(t *testing.T) {
	t.Parallel()

	ceilings := askCeilingsWithAPeerAtFourHundred(t)
	ceilings.AskCancelled(t.Context(), peerAddress, mostDocuments, 1250*time.Millisecond)

	if ceiling := ceilings.CeilingOf(t.Context(), peerAddress); ceiling != 400 {
		t.Fatalf(
			"ceiling after a cancelled ask that proved 1.25ms per document = %d, want 400",
			ceiling,
		)
	}
}

func TestAFailedAskDoublesThePace(t *testing.T) {
	t.Parallel()

	ceilings := askCeilingsWithAPeerAtFourHundred(t)
	ceilings.AskFailed(t.Context(), peerAddress)

	if ceiling := ceilings.CeilingOf(t.Context(), peerAddress); ceiling != 200 {
		t.Fatalf("ceiling after a failed ask = %d, want 200", ceiling)
	}
}

func TestAFailedAskOfAFreshAddressTakesTheLeast(t *testing.T) {
	t.Parallel()

	ceilings := askCeilings()
	ceilings.AskFailed(t.Context(), peerAddress)

	if ceiling := ceilings.CeilingOf(t.Context(), peerAddress); ceiling != leastDocuments {
		t.Fatalf("ceiling after a failed ask of a fresh address = %d, want %d",
			ceiling, leastDocuments)
	}
}

func TestTheFirstSampleSetsTheAverageAndLaterSamplesWeighHalf(t *testing.T) {
	t.Parallel()

	ceilings := askCeilingsWithAPeerAtFourHundred(t)
	ceilings.AskAnswered(t.Context(), peerAddress, 400, 2*time.Second)

	if ceiling := ceilings.CeilingOf(t.Context(), peerAddress); ceiling != 267 {
		t.Fatalf("ceiling after 2.5ms then 5ms per document = %d, want 267", ceiling)
	}
}

func TestTheCeilingStaysBetweenTheLeastAndTheMost(t *testing.T) {
	t.Parallel()

	ceilings := askCeilings()
	ceilings.AskAnswered(t.Context(), "http://fast.example", mostDocuments, 0)
	ceilings.AskAnswered(t.Context(), "http://slow.example", mostDocuments, 100*time.Second)

	if ceiling := ceilings.CeilingOf(t.Context(), "http://fast.example"); ceiling != mostDocuments {
		t.Fatalf("ceiling of a peer that answered at once = %d, want %d", ceiling, mostDocuments)
	}
	if ceiling := ceilings.CeilingOf(
		t.Context(),
		"http://slow.example",
	); ceiling != leastDocuments {
		t.Fatalf("ceiling of a peer that spent 100ms per document = %d, want %d",
			ceiling, leastDocuments)
	}
}

func TestTheLeastIsNeverAboveTheMost(t *testing.T) {
	t.Parallel()

	ceilings := urlmetadataaskceilings.New(
		10, leastDocuments, targetTime, peerPacesInAMap{}, &recordedCeilings{},
	)
	ceilings.AskFailed(t.Context(), peerAddress)

	if ceiling := ceilings.CeilingOf(t.Context(), peerAddress); ceiling != 10 {
		t.Fatalf("ceiling after a failed ask with the least above the most = %d, want 10", ceiling)
	}
}

func TestTheObserverSeesEachCeilingSet(t *testing.T) {
	t.Parallel()

	observer := &recordedCeilings{}
	ceilings := urlmetadataaskceilings.New(
		mostDocuments, leastDocuments, targetTime, peerPacesInAMap{}, observer,
	)
	ceilings.AskAnswered(t.Context(), "http://fast.example", mostDocuments, time.Second)
	ceilings.AskAnswered(t.Context(), peerAddress, mostDocuments, 2500*time.Millisecond)

	want := []setCeiling{
		{
			address: "http://fast.example",
			pace:    time.Millisecond,
			ceiling: mostDocuments,
		},
		{address: peerAddress, pace: 2500 * time.Microsecond, ceiling: 400},
	}
	if !slices.Equal(observer.setCeilings, want) {
		t.Fatalf("the observer saw %+v, want %+v", observer.setCeilings, want)
	}
}
