package distributioncycle_test

import (
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
)

const callMeetingWait = time.Second

func TestCycleOffersABatchToAllItsPeersAtTheSameTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		word, url := yacymodel.WordHash("w1"), urlHash("u1")
		peerA, peerB := yacymodel.WordHash("a"), yacymodel.WordHash("b")
		h := openCycle(t, &clock{at: time.Unix(1000, 0)}, cycleOptions{
			postings: map[yacymodel.Hash]yacymodel.RWIPosting{
				fakePostingKey(word, url): fakePosting(word, url),
			},
			roster:     fakeRoster{reachable: []yacymodel.Seed{seed(peerA), seed(peerB)}},
			redundancy: 2,
		})
		meeting := newCallMeeting(2)
		h.courier.onOffer = meeting.arrive

		store(t, h.v, h.schedule, word, url)

		h.runCycle(t)

		meeting.assertMet(t)
	})
}

func TestCycleOffersBatchesOfDifferentSectorsAtTheSameTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		peer := yacymodel.WordHash("peer")
		sectorPostings := append(
			postingsInSector(1, postingsPerBatch),
			postingsInSector(40, postingsPerBatch)...,
		)
		h := openCycle(t, &clock{at: time.Unix(1000, 0)}, cycleOptions{
			postings:         postingsByKey(sectorPostings),
			roster:           fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
			redundancy:       2,
			batchConcurrency: 2,
		})
		h.courier.receipts[peer] = postingcourier.Receipt{Outcome: postingcourier.Accepted}
		meeting := newCallMeeting(2)
		h.courier.onOffer = meeting.arrive

		for _, posting := range sectorPostings {
			store(t, h.v, h.schedule, posting.WordHash, posting.URLHash)
		}

		h.runCycle(t)

		meeting.assertMet(t)
		assertEachPostingOfferedOnce(t, h.courier.offered, sectorPostings)
	})
}

func TestCycleReadsTheDuePostingsOfBatchesAtTheSameTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		peer := yacymodel.WordHash("peer")
		sectorPostings := append(
			postingsInSector(1, postingsPerBatch),
			postingsInSector(40, postingsPerBatch)...,
		)
		h := openCycle(t, &clock{at: time.Unix(1000, 0)}, cycleOptions{
			postings:         postingsByKey(sectorPostings),
			roster:           fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
			redundancy:       2,
			batchConcurrency: 2,
		})
		h.courier.receipts[peer] = postingcourier.Receipt{Outcome: postingcourier.Accepted}
		meeting := newCallMeeting(2)
		h.postings.onRead = meeting.arrive

		for _, posting := range sectorPostings {
			store(t, h.v, h.schedule, posting.WordHash, posting.URLHash)
		}

		h.runCycle(t)

		meeting.assertMet(t)
	})
}

func postingsInSector(
	sector yacymodel.DHTRingSector,
	amountOfPostings int,
) []yacymodel.RWIPosting {
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		panic(err)
	}

	var postings []yacymodel.RWIPosting
	for attempt := 0; len(postings) < amountOfPostings; attempt++ {
		name := fmt.Sprintf("sector%d-%d", sector, attempt)
		posting := fakePosting(yacymodel.WordHash(name), urlHash(name))
		position := yacymodel.DHTRingPositionOfPosting(posting, partitions)
		if yacymodel.DHTRingSectorOf(position) == sector {
			postings = append(postings, posting)
		}
	}

	return postings
}

func postingsByKey(postings []yacymodel.RWIPosting) map[yacymodel.Hash]yacymodel.RWIPosting {
	byKey := make(map[yacymodel.Hash]yacymodel.RWIPosting, len(postings))
	for _, posting := range postings {
		byKey[fakePostingKey(posting.WordHash, posting.URLHash)] = posting
	}

	return byKey
}

func assertEachPostingOfferedOnce(
	t *testing.T,
	offered []offeredCall,
	postings []yacymodel.RWIPosting,
) {
	t.Helper()

	offers := make(map[yacymodel.RWIPosting]int)
	for _, call := range offered {
		for _, posting := range call.Postings {
			offers[posting]++
		}
	}
	for _, posting := range postings {
		if offers[posting] != 1 {
			t.Fatalf("offers = %v, want every posting offered once", offers)
		}
	}
}

type callMeeting struct {
	guard       sync.Mutex
	awaited     int
	arrived     int
	allArrived  chan struct{}
	waitedAlone bool
}

func newCallMeeting(awaited int) *callMeeting {
	return &callMeeting{awaited: awaited, allArrived: make(chan struct{})}
}

func (m *callMeeting) arrive() {
	m.guard.Lock()
	m.arrived++
	if m.arrived == m.awaited {
		close(m.allArrived)
	}
	m.guard.Unlock()

	select {
	case <-m.allArrived:
	case <-time.After(callMeetingWait):
		m.guard.Lock()
		m.waitedAlone = true
		m.guard.Unlock()
	}
}

func (m *callMeeting) assertMet(t *testing.T) {
	t.Helper()

	m.guard.Lock()
	defer m.guard.Unlock()

	if m.waitedAlone || m.arrived < m.awaited {
		t.Fatalf("%d calls arrived, want %d of them at the same time", m.arrived, m.awaited)
	}
}
