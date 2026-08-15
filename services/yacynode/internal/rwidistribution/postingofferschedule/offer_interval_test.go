package postingofferschedule_test

import (
	"testing"
	"time"
)

func openOffersWithPosting(t *testing.T) *postingOffers {
	t.Helper()

	offers := openOffers(t, testStart)
	offers.store(t, testWord, urlHash("u1"))

	return offers
}

func missRedundancy(
	t *testing.T,
	offers *postingOffers,
	requestedPause time.Duration,
) time.Duration {
	t.Helper()

	offers.clock = testStart
	offers.pauseOffer(t, testWord, urlHash("u1"), requestedPause)

	return nextOfferIn(t, offers)
}

func meetRedundancyAt(t *testing.T, offers *postingOffers, now time.Time) time.Time {
	t.Helper()

	offers.clock = now
	offers.meetRedundancy(t, testWord, urlHash("u1"))

	return testStart.Add(nextOfferIn(t, offers))
}

func nextOfferIn(t *testing.T, offers *postingOffers) time.Duration {
	t.Helper()

	low, high := time.Duration(0), 32*time.Minute
	for low < high {
		middle := low + (high-low)/2/time.Second*time.Second
		if middle == low {
			break
		}
		if offers.duePostingsAt(t, testStart.Add(middle)) > 0 {
			high = middle
		} else {
			low = middle
		}
	}

	return high
}

func TestFirstMissUsesTheShortestOfferInterval(t *testing.T) {
	offers := openOffersWithPosting(t)

	if next := missRedundancy(t, offers, 0); next != testInterval.Shortest {
		t.Fatalf("next offer in %v, want %v on the first miss", next, testInterval.Shortest)
	}
}

func TestFurtherMissesDoubleTheIntervalUpToTheLongest(t *testing.T) {
	offers := openOffersWithPosting(t)

	for _, want := range []time.Duration{
		time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 8 * time.Minute,
	} {
		if next := missRedundancy(t, offers, 0); next != want {
			t.Fatalf("next offer in %v, want %v", next, want)
		}
	}
}

func TestAMissUsesTheRequestedPauseWhenItIsLonger(t *testing.T) {
	offers := openOffersWithPosting(t)

	if next := missRedundancy(t, offers, 5*time.Minute); next != 5*time.Minute {
		t.Fatalf("next offer in %v, want the requested pause of %v", next, 5*time.Minute)
	}
}

func TestMetRedundancyForgetsTheWidenedInterval(t *testing.T) {
	offers := openOffersWithPosting(t)

	missRedundancy(t, offers, 0)
	missRedundancy(t, offers, 0)
	meetRedundancyAt(t, offers, testStart)

	if next := missRedundancy(t, offers, 0); next != testInterval.Shortest {
		t.Fatalf("next offer in %v, want %v once redundancy was met", next, testInterval.Shortest)
	}
}

func TestPurgedPostingReturnsToTheShortestOfferInterval(t *testing.T) {
	offers := openOffersWithPosting(t)

	missRedundancy(t, offers, 0)
	offers.purge(t, testWord, urlHash("u1"))
	offers.store(t, testWord, urlHash("u1"))

	if next := missRedundancy(t, offers, 0); next != testInterval.Shortest {
		t.Fatalf(
			"next offer in %v, want %v after the posting was purged",
			next,
			testInterval.Shortest,
		)
	}
}

func TestAMissDoesNotScheduleAnUnscheduledPosting(t *testing.T) {
	offers := openOffers(t, testStart)

	missRedundancy(t, offers, 0)

	if due := offers.duePostingsAt(t, testStart.Add(time.Hour)); due != 0 {
		t.Fatalf("due postings = %d, want 0 for an unscheduled posting", due)
	}
}

func TestMetRedundancyAnchorsTheNextOfferAtThePreviousDueTime(t *testing.T) {
	offers := openOffersWithPosting(t)
	lateness := 3 * time.Minute

	dueAt := meetRedundancyAt(t, offers, testStart.Add(lateness))

	if want := testStart.Add(testInterval.Longest); !dueAt.Equal(want) {
		t.Fatalf("next offer due at %v, want %v anchored at the previous due time", dueAt, want)
	}
}

func TestMetRedundancySkipsMissedOfferGenerations(t *testing.T) {
	offers := openOffersWithPosting(t)
	now := testStart.Add(2*testInterval.Longest + time.Minute)

	dueAt := meetRedundancyAt(t, offers, now)

	if want := testStart.Add(3 * testInterval.Longest); !dueAt.Equal(want) {
		t.Fatalf("next offer due at %v, want %v skipping the missed offers", dueAt, want)
	}
	if !dueAt.After(now) {
		t.Fatalf("next offer due at %v, want a time after %v", dueAt, now)
	}
}

func TestMetRedundancyMovesPastTheExactIntervalBoundary(t *testing.T) {
	offers := openOffersWithPosting(t)
	now := testStart.Add(testInterval.Longest)

	dueAt := meetRedundancyAt(t, offers, now)

	if want := testStart.Add(2 * testInterval.Longest); !dueAt.Equal(want) {
		t.Fatalf("next offer due at %v, want %v on the interval boundary", dueAt, want)
	}
}
