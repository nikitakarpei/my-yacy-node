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

func (o *postingOffers) meetRedundancy(t *testing.T) time.Duration {
	t.Helper()

	o.clock = testStart
	if err := o.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return o.schedule.SetNextOfferAfterRedundancyMet(
			tx, postingidentity.IdentityOf(testWord, urlHash("u1")), testInterval,
		)
	}); err != nil {
		t.Fatalf("SetNextOfferAfterRedundancyMet: %v", err)
	}

	return o.nextOfferIn(t)
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

func TestMetRedundancyUsesTheLongestOfferIntervalAndForgetsTheWidenedOne(t *testing.T) {
	offers := openOffers(t)

	offers.missRedundancy(t, 0)
	offers.missRedundancy(t, 0)

	if next := offers.meetRedundancy(t); next != testInterval.Longest {
		t.Fatalf("next offer in %v, want %v", next, testInterval.Longest)
	}
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
