package postingofferschedule

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

var testInterval = OfferInterval{Shortest: time.Minute, Longest: 8 * time.Minute}

var testStart = time.Unix(1_000_000, 0).UTC()

var testWord = yacymodel.WordHash("w1")

type postingOffers struct {
	vault    *vault.Vault
	schedule *Schedule
	clock    time.Time
}

func openOffers(t *testing.T) *postingOffers {
	t.Helper()

	offers := openOffersWithoutPosting(t)
	store(t, offers.schedule, testWord, urlHash("u1"))

	return offers
}

func openOffersWithoutPosting(t *testing.T) *postingOffers {
	t.Helper()

	offers := &postingOffers{clock: testStart}
	offers.vault, offers.schedule, _ = openSchedule(
		t, func() time.Time { return offers.clock },
	)

	return offers
}

func (o *postingOffers) missRedundancy(t *testing.T, requestedPause time.Duration) time.Duration {
	t.Helper()

	o.clock = testStart
	if err := o.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return o.schedule.SetNextOfferAfterRedundancyMissed(
			tx, postingidentity.IdentityOf(testWord, urlHash("u1")), testInterval, requestedPause,
		)
	}); err != nil {
		t.Fatalf("SetNextOfferAfterRedundancyMissed: %v", err)
	}

	return o.nextOfferIn(t)
}

func (o *postingOffers) meetRedundancyAt(t *testing.T, now time.Time) time.Time {
	t.Helper()

	o.clock = now
	posting := postingidentity.IdentityOf(testWord, urlHash("u1"))
	var dueAt time.Time
	if err := o.vault.Update(context.Background(), func(tx *vault.Txn) error {
		if err := o.schedule.SetNextOfferAfterRedundancyMet(tx, posting, testInterval); err != nil {
			return err
		}
		var err error
		dueAt, _, err = o.schedule.dueAt(tx, posting)

		return err
	}); err != nil {
		t.Fatalf("SetNextOfferAfterRedundancyMet: %v", err)
	}

	return dueAt
}

func (o *postingOffers) nextOfferIn(t *testing.T) time.Duration {
	t.Helper()

	low, high := time.Duration(0), 32*time.Minute
	for low < high {
		middle := low + (high-low)/2/time.Second*time.Second
		if middle == low {
			break
		}
		if o.duePostings(t, testStart.Add(middle)) > 0 {
			high = middle
		} else {
			low = middle
		}
	}

	return high
}

func (o *postingOffers) duePostings(t *testing.T, at time.Time) int {
	t.Helper()

	o.clock = at
	due, err := o.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}

	return len(due)
}

func TestFirstMissUsesTheShortestOfferInterval(t *testing.T) {
	offers := openOffers(t)

	if next := offers.missRedundancy(t, 0); next != testInterval.Shortest {
		t.Fatalf("next offer in %v, want %v on the first miss", next, testInterval.Shortest)
	}
}

func TestFurtherMissesDoubleTheIntervalUpToTheLongest(t *testing.T) {
	offers := openOffers(t)

	for _, want := range []time.Duration{
		time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 8 * time.Minute,
	} {
		if next := offers.missRedundancy(t, 0); next != want {
			t.Fatalf("next offer in %v, want %v", next, want)
		}
	}
}

func TestAMissUsesTheRequestedPauseWhenItIsLonger(t *testing.T) {
	offers := openOffers(t)

	if next := offers.missRedundancy(t, 5*time.Minute); next != 5*time.Minute {
		t.Fatalf("next offer in %v, want the requested pause of %v", next, 5*time.Minute)
	}
}

func TestMetRedundancyForgetsTheWidenedInterval(t *testing.T) {
	offers := openOffers(t)

	offers.missRedundancy(t, 0)
	offers.missRedundancy(t, 0)
	offers.meetRedundancyAt(t, testStart)

	if next := offers.missRedundancy(t, 0); next != testInterval.Shortest {
		t.Fatalf("next offer in %v, want %v once redundancy was met", next, testInterval.Shortest)
	}
}

func TestPurgedPostingReturnsToTheShortestOfferInterval(t *testing.T) {
	offers := openOffers(t)

	offers.missRedundancy(t, 0)
	if err := offers.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return offers.schedule.PostingPurged(tx, testWord, urlHash("u1"))
	}); err != nil {
		t.Fatalf("PostingPurged: %v", err)
	}
	store(t, offers.schedule, testWord, urlHash("u1"))

	if next := offers.missRedundancy(t, 0); next != testInterval.Shortest {
		t.Fatalf(
			"next offer in %v, want %v after the posting was purged",
			next,
			testInterval.Shortest,
		)
	}
}

func TestAMissDoesNotScheduleAnUnscheduledPosting(t *testing.T) {
	offers := openOffersWithoutPosting(t)

	offers.missRedundancy(t, 0)

	if due := offers.duePostings(t, testStart.Add(time.Hour)); due != 0 {
		t.Fatalf("due postings = %d, want 0 for an unscheduled posting", due)
	}
}

func TestMetRedundancyAnchorsTheNextOfferAtThePreviousDueTime(t *testing.T) {
	offers := openOffers(t)
	lateness := 3 * time.Minute

	dueAt := offers.meetRedundancyAt(t, testStart.Add(lateness))

	if want := testStart.Add(testInterval.Longest); !dueAt.Equal(want) {
		t.Fatalf("next offer due at %v, want %v anchored at the previous due time", dueAt, want)
	}
}

func TestMetRedundancySkipsMissedOfferGenerations(t *testing.T) {
	offers := openOffers(t)
	now := testStart.Add(2*testInterval.Longest + time.Minute)

	dueAt := offers.meetRedundancyAt(t, now)

	if want := testStart.Add(3 * testInterval.Longest); !dueAt.Equal(want) {
		t.Fatalf("next offer due at %v, want %v skipping the missed offers", dueAt, want)
	}
	if !dueAt.After(now) {
		t.Fatalf("next offer due at %v, want a time after %v", dueAt, now)
	}
}

func TestMetRedundancyMovesPastTheExactIntervalBoundary(t *testing.T) {
	offers := openOffers(t)
	now := testStart.Add(testInterval.Longest)

	dueAt := offers.meetRedundancyAt(t, now)

	if want := testStart.Add(2 * testInterval.Longest); !dueAt.Equal(want) {
		t.Fatalf("next offer due at %v, want %v on the interval boundary", dueAt, want)
	}
}
