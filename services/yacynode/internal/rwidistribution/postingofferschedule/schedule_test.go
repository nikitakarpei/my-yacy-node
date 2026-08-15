package postingofferschedule_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestPostingStoredIsImmediatelyDue(t *testing.T) {
	offers := openOffers(t, testStart)
	word, url := testWord, urlHash("u1")
	offers.store(t, word, url)

	due := offers.duePostings(t, 10)
	if len(due) != 1 || due[0].Word != word || due[0].URL != url {
		t.Fatalf("due = %v, want single entry for %v/%v", due, word, url)
	}
}

func TestDuePostingsExcludesPausedEntries(t *testing.T) {
	offers := openOffers(t, testStart)
	overdue, paused := yacymodel.WordHash("overdue"), yacymodel.WordHash("paused")
	url := urlHash("u1")
	offers.store(t, overdue, url)
	offers.store(t, paused, url)

	offers.pauseOffer(t, paused, url, time.Hour)

	due := offers.duePostings(t, 10)
	if len(due) != 1 || due[0].Word != overdue {
		t.Fatalf("due = %v, want only [overdue]", due)
	}
}

func TestDuePostingsRespectsLimit(t *testing.T) {
	offers := openOffers(t, testStart)
	for _, seed := range []string{"a", "b", "c"} {
		offers.store(t, yacymodel.WordHash(seed), urlHash(seed))
	}

	if due := offers.duePostings(t, 2); len(due) != 2 {
		t.Fatalf("due = %v, want 2 entries", due)
	}
}

func TestPostingPurgedLeavesNothingScheduled(t *testing.T) {
	offers := openOffers(t, testStart)
	word, url := testWord, urlHash("u1")
	offers.store(t, word, url)

	offers.purge(t, word, url)

	if due := offers.duePostings(t, 10); len(due) != 0 {
		t.Fatalf("due = %v, want none after purge", due)
	}
	if offers.isScheduled(t, word, url) {
		t.Fatal("purged posting is still scheduled")
	}
}

func TestPausedOfferDoesNotResurrectPurgedPosting(t *testing.T) {
	offers := openOffers(t, testStart)
	word, url := testWord, urlHash("u1")
	offers.store(t, word, url)

	offers.purge(t, word, url)
	offers.pauseOffer(t, word, url, time.Hour)

	if offers.isScheduled(t, word, url) {
		t.Fatal("purged posting came back when its offer was paused")
	}
}

func TestPostingPurgedUnknownIsHarmless(t *testing.T) {
	offers := openOffers(t, testStart)

	offers.purge(t, yacymodel.WordHash("absent"), urlHash("absent"))
}

func TestMetRedundancyForgetsTheWidenedInterval(t *testing.T) {
	offers := openOffersWithPosting(t)

	missRedundancy(t, offers)
	missRedundancy(t, offers)
	offers.clock = testStart
	offers.meetRedundancy(t, testWord, urlHash("u1"))

	if next := missRedundancy(t, offers); next != testInterval.Shortest {
		t.Fatalf("next offer in %v, want %v once redundancy was met", next, testInterval.Shortest)
	}
}

func openOffersWithPosting(t *testing.T) *postingOffers {
	t.Helper()

	offers := openOffers(t, testStart)
	offers.store(t, testWord, urlHash("u1"))

	return offers
}

func missRedundancy(t *testing.T, offers *postingOffers) time.Duration {
	t.Helper()

	offers.clock = testStart
	offers.pauseOffer(t, testWord, urlHash("u1"), 0)

	return nextOfferIn(t, offers)
}

func nextOfferIn(t *testing.T, offers *postingOffers) time.Duration {
	t.Helper()

	wellAfterAnyOffer := 24 * time.Hour
	offers.clock = testStart.Add(wellAfterAnyOffer)
	offers.schedule.ObserveBacklog(t.Context())

	return wellAfterAnyOffer - offers.observed.lateness
}

func TestPurgedPostingReturnsToTheShortestOfferInterval(t *testing.T) {
	offers := openOffersWithPosting(t)

	missRedundancy(t, offers)
	offers.purge(t, testWord, urlHash("u1"))
	offers.store(t, testWord, urlHash("u1"))

	if next := missRedundancy(t, offers); next != testInterval.Shortest {
		t.Fatalf(
			"next offer in %v, want %v after the posting was purged",
			next,
			testInterval.Shortest,
		)
	}
}

func TestAMissDoesNotScheduleAnUnscheduledPosting(t *testing.T) {
	offers := openOffers(t, testStart)

	missRedundancy(t, offers)

	if due := offers.duePostingsAt(t, testStart.Add(time.Hour)); due != 0 {
		t.Fatalf("due postings = %d, want 0 for an unscheduled posting", due)
	}
}

func TestObserveCountsScheduledPostings(t *testing.T) {
	offers := openOffers(t, testStart)

	offers.schedule.ObserveBacklog(t.Context())
	if offers.observed.scheduled != 0 {
		t.Fatalf("scheduled = %d, want 0 for an empty schedule", offers.observed.scheduled)
	}

	offers.store(t, testWord, urlHash("u1"))

	offers.schedule.ObserveBacklog(t.Context())
	if offers.observed.scheduled != 1 {
		t.Fatalf("scheduled = %d, want 1 after a posting is stored", offers.observed.scheduled)
	}
}

func TestObserveReportsNoLatenessForEmptySchedule(t *testing.T) {
	offers := openOffers(t, testStart)

	offers.schedule.ObserveBacklog(t.Context())

	if offers.observed.lateness != 0 {
		t.Fatalf("lateness = %v, want 0 for an empty schedule", offers.observed.lateness)
	}
}

func TestObserveReportsLatenessOfEarliestEntry(t *testing.T) {
	offers := openOffers(t, testStart)
	url := urlHash("u1")
	offers.store(t, yacymodel.WordHash("earlier"), url)

	offers.clock = testStart.Add(time.Hour)
	offers.store(t, yacymodel.WordHash("later"), url)

	offers.schedule.ObserveBacklog(t.Context())

	if offers.observed.lateness != time.Hour {
		t.Fatalf("lateness = %v, want %v", offers.observed.lateness, time.Hour)
	}
}
