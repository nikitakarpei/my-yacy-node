package postingofferschedule_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestPostingStoredIsImmediatelyDue(t *testing.T) {
	schedule := openSchedule(t, testStart)
	word, url := testWord, urlHash("u1")
	schedule.store(t, word, url)

	due := schedule.duePostings(t, 10)
	if len(due) != 1 || due[0].Word != word || due[0].URL != url {
		t.Fatalf("due = %v, want single entry for %v/%v", due, word, url)
	}
}

func TestDuePostingsExcludesPausedEntries(t *testing.T) {
	schedule := openSchedule(t, testStart)
	overdue, paused := yacymodel.WordHash("overdue"), yacymodel.WordHash("paused")
	url := urlHash("u1")
	schedule.store(t, overdue, url)
	schedule.store(t, paused, url)

	schedule.pauseOffer(t, paused, url, time.Hour)

	due := schedule.duePostings(t, 10)
	if len(due) != 1 || due[0].Word != overdue {
		t.Fatalf("due = %v, want only [overdue]", due)
	}
}

func TestDuePostingsRespectsLimit(t *testing.T) {
	schedule := openSchedule(t, testStart)
	for _, seed := range []string{"a", "b", "c"} {
		schedule.store(t, yacymodel.WordHash(seed), urlHash(seed))
	}

	if due := schedule.duePostings(t, 2); len(due) != 2 {
		t.Fatalf("due = %v, want 2 entries", due)
	}
}

func TestAPostingPurgedAndStoredAgainIsDueAgain(t *testing.T) {
	schedule := openSchedule(t, testStart)
	word, url := testWord, urlHash("u1")
	schedule.store(t, word, url)
	schedule.pauseOffer(t, word, url, time.Hour)

	if due := schedule.duePostings(t, 10); len(due) != 0 {
		t.Fatalf("due = %v, want none while the offer is paused", due)
	}

	schedule.purge(t, word, url)
	schedule.store(t, word, url)

	due := schedule.duePostings(t, 10)
	if len(due) != 1 || due[0].Word != word || due[0].URL != url {
		t.Fatalf("due = %v, want single entry for %v/%v", due, word, url)
	}
}

func TestPostingPurgedLeavesNothingScheduled(t *testing.T) {
	schedule := openSchedule(t, testStart)
	word, url := testWord, urlHash("u1")
	schedule.store(t, word, url)

	schedule.purge(t, word, url)

	if due := schedule.duePostings(t, 10); len(due) != 0 {
		t.Fatalf("due = %v, want none after purge", due)
	}
	if schedule.isScheduled(t, word, url) {
		t.Fatal("purged posting is still scheduled")
	}
}

func TestPausedOfferDoesNotResurrectPurgedPosting(t *testing.T) {
	schedule := openSchedule(t, testStart)
	word, url := testWord, urlHash("u1")
	schedule.store(t, word, url)

	schedule.purge(t, word, url)
	schedule.pauseOffer(t, word, url, time.Hour)

	if schedule.isScheduled(t, word, url) {
		t.Fatal("purged posting came back when its offer was paused")
	}
}

func TestPostingPurgedUnknownIsHarmless(t *testing.T) {
	schedule := openSchedule(t, testStart)

	schedule.purge(t, yacymodel.WordHash("absent"), urlHash("absent"))
}

func TestMetRedundancyForgetsTheWidenedInterval(t *testing.T) {
	schedule := openScheduleWithPosting(t)

	missRedundancyAtStart(t, schedule)
	missRedundancyAtStart(t, schedule)
	schedule.clock = testStart
	schedule.meetRedundancy(t, testWord, urlHash("u1"))
	missRedundancyAtStart(t, schedule)

	if next := nextShortfallOfferIn(t, schedule); next != testInterval.Shortest {
		t.Fatalf("next offer in %v, want %v once redundancy was met", next, testInterval.Shortest)
	}
}

func openScheduleWithPosting(t *testing.T) *scheduleHarness {
	t.Helper()

	schedule := openSchedule(t, testStart)
	schedule.store(t, testWord, urlHash("u1"))

	return schedule
}

func missRedundancyAtStart(t *testing.T, schedule *scheduleHarness) {
	t.Helper()

	schedule.clock = testStart
	schedule.pauseOffer(t, testWord, urlHash("u1"), 0)
}

func nextShortfallOfferIn(t *testing.T, schedule *scheduleHarness) time.Duration {
	t.Helper()

	wellAfterAnyOffer := 24 * time.Hour
	schedule.clock = testStart.Add(wellAfterAnyOffer)
	schedule.observeBacklog(t)

	return wellAfterAnyOffer - schedule.observed.lateness[shortfall]
}

func TestPurgedPostingReturnsToTheShortestOfferInterval(t *testing.T) {
	schedule := openScheduleWithPosting(t)

	missRedundancyAtStart(t, schedule)
	schedule.purge(t, testWord, urlHash("u1"))
	schedule.store(t, testWord, urlHash("u1"))
	missRedundancyAtStart(t, schedule)

	if next := nextShortfallOfferIn(t, schedule); next != testInterval.Shortest {
		t.Fatalf(
			"next offer in %v, want %v after the posting was purged",
			next,
			testInterval.Shortest,
		)
	}
}

func TestAMissDoesNotScheduleAnUnscheduledPosting(t *testing.T) {
	schedule := openSchedule(t, testStart)

	missRedundancyAtStart(t, schedule)

	schedule.clock = testStart.Add(time.Hour)
	if due := schedule.duePostings(t, 10); len(due) != 0 {
		t.Fatalf("due = %v, want none for an unscheduled posting", due)
	}
}

func TestObserveCountsScheduledPostings(t *testing.T) {
	schedule := openSchedule(t, testStart)

	schedule.observeBacklog(t)
	if schedule.observed.scheduled[shortfall] != 0 || schedule.observed.scheduled[refresh] != 0 {
		t.Fatalf("scheduled = %v, want 0 in each order for an empty schedule",
			schedule.observed.scheduled)
	}

	schedule.store(t, testWord, urlHash("u1"))
	schedule.store(t, testWord, urlHash("u2"))
	schedule.meetRedundancy(t, testWord, urlHash("u2"))

	schedule.observeBacklog(t)
	if schedule.observed.scheduled[shortfall] != 1 || schedule.observed.scheduled[refresh] != 1 {
		t.Fatalf("scheduled = %v, want 1 short of redundancy and 1 at it",
			schedule.observed.scheduled)
	}
}

func TestObserveReportsNoLatenessForEmptySchedule(t *testing.T) {
	schedule := openSchedule(t, testStart)

	schedule.observeBacklog(t)

	if schedule.observed.lateness[shortfall] != 0 || schedule.observed.lateness[refresh] != 0 {
		t.Fatalf("lateness = %v, want 0 in each order for an empty schedule",
			schedule.observed.lateness)
	}
}

func TestObserveReportsLatenessOfEarliestEntry(t *testing.T) {
	schedule := openSchedule(t, testStart)
	url := urlHash("u1")
	schedule.store(t, yacymodel.WordHash("earlier"), url)

	schedule.clock = testStart.Add(time.Hour)
	schedule.store(t, yacymodel.WordHash("later"), url)

	schedule.observeBacklog(t)

	if schedule.observed.lateness[shortfall] != time.Hour {
		t.Fatalf(
			"shortfall lateness = %v, want %v",
			schedule.observed.lateness[shortfall],
			time.Hour,
		)
	}
}

func TestDuePostingsShortOfRedundancyComeBeforeRefreshes(t *testing.T) {
	schedule := openSchedule(t, testStart)
	refreshed, stored := urlHash("u1"), urlHash("u2")
	schedule.store(t, testWord, refreshed)
	schedule.meetRedundancy(t, testWord, refreshed)
	schedule.clock = testStart.Add(2 * testInterval.Longest)
	schedule.store(t, testWord, stored)

	due := schedule.duePostings(t, 1)

	if len(due) != 1 || due[0].URL != stored {
		t.Fatalf("due = %v, want only the posting short of redundancy", due)
	}
}

func TestDuePostingsFillTheRestOfTheBatchFromRefreshes(t *testing.T) {
	schedule := openSchedule(t, testStart)
	refreshed, stored := urlHash("u1"), urlHash("u2")
	schedule.store(t, testWord, refreshed)
	schedule.meetRedundancy(t, testWord, refreshed)
	schedule.clock = testStart.Add(2 * testInterval.Longest)
	schedule.store(t, testWord, stored)

	due := schedule.duePostings(t, 10)

	if len(due) != 2 || due[0].URL != stored || due[1].URL != refreshed {
		t.Fatalf("due = %v, want the short posting, then the refresh", due)
	}
}

func TestMissedRedundancyMovesARefreshBackToTheShortfall(t *testing.T) {
	schedule := openSchedule(t, testStart)
	url := urlHash("u1")
	schedule.store(t, testWord, url)
	schedule.meetRedundancy(t, testWord, url)
	schedule.pauseOffer(t, testWord, url, 0)

	schedule.observeBacklog(t)

	if schedule.observed.scheduled[shortfall] != 1 || schedule.observed.scheduled[refresh] != 0 {
		t.Fatalf("scheduled = %v, want the posting back short of redundancy",
			schedule.observed.scheduled)
	}
}

func TestPurgedRefreshLeavesNothingScheduled(t *testing.T) {
	schedule := openSchedule(t, testStart)
	url := urlHash("u1")
	schedule.store(t, testWord, url)
	schedule.meetRedundancy(t, testWord, url)
	schedule.purge(t, testWord, url)

	schedule.observeBacklog(t)

	if schedule.observed.scheduled[shortfall] != 0 || schedule.observed.scheduled[refresh] != 0 {
		t.Fatalf("scheduled = %v, want nothing left in either order", schedule.observed.scheduled)
	}
}
