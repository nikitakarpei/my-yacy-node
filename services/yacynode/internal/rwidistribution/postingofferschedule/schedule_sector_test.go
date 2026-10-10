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

func TestABatchTakesTheDuePostingsOfItsSectorOnly(t *testing.T) {
	schedule := openSchedule(t, testStart)
	sectorWords, nextSector := wordsIn(5, 2), wordIn(6)
	schedule.storeInTurn(t, sectorWords[0], nextSector, sectorWords[1])

	due := schedule.duePostingsIn(t, 5, 10)

	assertWordsInOrder(t, due, sectorWords...)
}

func TestRefreshesFillTheBatchOfTheirSector(t *testing.T) {
	schedule := openSchedule(t, testStart)
	sectorWords, nextSector := wordsIn(20, 2), wordIn(21)
	refresh, shortfallWord := sectorWords[0], sectorWords[1]
	schedule.storeInTurn(t, refresh, nextSector)
	schedule.meetRedundancy(t, refresh, urlHash("u1"))
	schedule.meetRedundancy(t, nextSector, urlHash("u1"))
	schedule.clock = testStart.Add(2 * testInterval.Longest)
	schedule.store(t, shortfallWord, urlHash("u1"))

	due := schedule.duePostingsIn(t, 20, 10)

	assertWordsInOrder(t, due, shortfallWord, refresh)
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

func TestTheEarliestDueSectorSkipsTheSectorsOfBatchesInFlight(t *testing.T) {
	schedule := openSchedule(t, testStart)
	schedule.storeInTurn(t, wordIn(4), wordIn(6), wordIn(5))

	sector := schedule.earliestDueSectorOutside(t, []yacymodel.DHTRingSector{4})

	if sector != 6 {
		t.Fatalf("sector = %d, want 6, the earliest due outside sector 4", sector)
	}
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
