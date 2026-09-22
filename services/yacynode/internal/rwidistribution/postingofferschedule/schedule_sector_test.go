package postingofferschedule_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
)

func TestABatchTakesTheSectorOfTheEarliestDuePostingFirst(t *testing.T) {
	schedule := openSchedule(t, testStart)
	sectorWords, otherSector := wordsIn(5, 2), wordIn(2)
	earliest, sameSector := sectorWords[0], sectorWords[1]
	schedule.storeInTurn(t, earliest, otherSector, sameSector)

	due := schedule.duePostings(t, 2)

	assertWordsInOrder(t, due, earliest, sameSector)
}

func TestABatchWalksOnFromTheLastSectorToTheFirst(t *testing.T) {
	schedule := openSchedule(t, testStart)
	lastSector, middleSector, firstSector := wordIn(
		yacymodel.MaxDHTRingSector,
	), wordIn(
		10,
	), wordIn(
		0,
	)
	schedule.storeInTurn(t, lastSector, middleSector, firstSector)

	due := schedule.duePostings(t, 2)

	assertWordsInOrder(t, due, lastSector, firstSector)
}

func TestRefreshesFillTheBatchFromTheSectorWhereItStarts(t *testing.T) {
	schedule := openSchedule(t, testStart)
	refreshBefore, refreshAfter := wordIn(3), wordIn(21)
	schedule.storeInTurn(t, refreshBefore, refreshAfter)
	schedule.meetRedundancy(t, refreshBefore, urlHash("u1"))
	schedule.meetRedundancy(t, refreshAfter, urlHash("u1"))
	schedule.clock = testStart.Add(2 * testInterval.Longest)
	shortfall := wordIn(20)
	schedule.store(t, shortfall, urlHash("u1"))

	due := schedule.duePostings(t, 2)

	assertWordsInOrder(t, due, shortfall, refreshAfter)
}

func TestABatchStartsAtTheEarliestRefreshWhileNoShortfallIsDue(t *testing.T) {
	schedule := openSchedule(t, testStart)
	paused, earlierRefresh, laterRefresh := wordIn(1), wordIn(30), wordIn(3)
	schedule.storeInTurn(t, earlierRefresh, laterRefresh)
	schedule.meetRedundancy(t, earlierRefresh, urlHash("u1"))
	schedule.meetRedundancy(t, laterRefresh, urlHash("u1"))
	schedule.clock = testStart.Add(2 * testInterval.Longest)
	schedule.store(t, paused, urlHash("u1"))
	schedule.pauseOffer(t, paused, urlHash("u1"), time.Hour)

	due := schedule.duePostings(t, 1)

	assertWordsInOrder(t, due, earlierRefresh)
}

func TestAPostingIsClearedFromItsSectorAfterThePartitionsChange(t *testing.T) {
	schedule := openSchedule(t, testStart)
	schedule.store(t, testWord, urlHash("u1"))

	schedule.reopenWith(t, yacymodel.DHTRingPartitions(1)<<6)
	schedule.purge(t, testWord, urlHash("u1"))

	schedule.observeBacklog(t)
	if schedule.observed.scheduled[shortfall] != 0 {
		t.Fatalf("scheduled = %v, want nothing left after the purge", schedule.observed.scheduled)
	}
}

func (o *scheduleHarness) storeInTurn(t *testing.T, words ...yacymodel.Hash) {
	t.Helper()

	for _, word := range words {
		o.store(t, word, urlHash("u1"))
		o.clock = o.clock.Add(time.Minute)
	}
}

func assertWordsInOrder(
	t *testing.T,
	due []postingidentity.Identity,
	words ...yacymodel.Hash,
) {
	t.Helper()

	if len(due) != len(words) {
		t.Fatalf("due = %v, want the words %v", due, words)
	}
	for position, word := range words {
		if due[position].Word != word {
			t.Fatalf("due = %v, want the words %v", due, words)
		}
	}
}
