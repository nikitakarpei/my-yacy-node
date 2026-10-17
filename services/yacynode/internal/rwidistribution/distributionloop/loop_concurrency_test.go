package distributionloop_test

import (
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const callMeetingWait = time.Second

func TestLoopOffersABatchToAllItsPeersAtTheSameTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		h := openLoop(t, startClock(), loopOptions{redundancy: 3})
		h.reach(peerAfter(wordPartition, 1), peerAfter(wordPartition, 2))
		meeting := newCallMeeting(2)
		h.transfers.onOffer = func(yacymodel.Seed) { meeting.arrive() }
		h.admit(t, wordPartition, urlsOf(1)...)

		h.start(t)

		meeting.assertMet(t)
	})
}

func TestLoopOffersTwoBatchesAtTheSameTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, wordPartitions := openTwoBatches(t)
		meeting := newCallMeeting(2)
		h.transfers.onOffer = func(yacymodel.Seed) { meeting.arrive() }

		h.start(t)

		meeting.assertMet(t)
		assertEachWordPartitionOfferedOnce(t, h.transfers.offered, wordPartitions)
	})
}

func TestLoopOffersOtherBatchesWhileABatchWaitsForASlowPeer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		slowWordPartition, fastWordPartition := wordPartitionsIn(1, 1)[0], wordPartitionsIn(40, 2)
		slowPeer, fastPeer := peerAfter(slowWordPartition, 1), peerAfter(fastWordPartition[1], 1)
		h := openLoop(t, startClock(), loopOptions{
			self:             thisNodeFartherThan(slowWordPartition),
			amountOfWorkers:  2,
			postingsPerBatch: 1,
		})
		h.reach(slowPeer, fastPeer)
		h.transfers.answers[slowPeer.Hash] = acceptedAnswer
		h.transfers.answers[fastPeer.Hash] = acceptedAnswer
		slowAnswer := make(chan struct{})
		h.transfers.onOffer = func(peer yacymodel.Seed) {
			if peer.Hash == slowPeer.Hash {
				<-slowAnswer
			}
		}
		h.admit(t, slowWordPartition, urlsOf(1)...)
		h.admit(t, fastWordPartition[0], urlsOf(1)...)
		h.start(t)

		h.admit(t, fastWordPartition[1], urlsOf(1)...)
		h.settle()

		if got := h.holdersOf(
			t,
			fastWordPartition[1],
		); !slices.Equal(
			got,
			[]yacymodel.Hash{fastPeer.Hash},
		) {
			t.Fatalf("holders = %v, want %v while the slow batch waits", got, fastPeer.Hash)
		}
		if got := h.amountOfUndeliveredPostings(t); got != 1 {
			t.Fatalf("undelivered postings = %d, want only the slow batch left", got)
		}
		close(slowAnswer)
		synctest.Wait()

		if got := h.amountOfUndeliveredPostings(t); got != 0 {
			t.Fatalf("undelivered postings = %d, want none once the slow peer answered", got)
		}
	})
}

func TestLoopOffersAtMostTheConcurrentBatchesItIsAllowed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartitions := []yacymodel.WordPartition{
			wordPartitionsIn(1, 1)[0], wordPartitionsIn(20, 1)[0], wordPartitionsIn(40, 1)[0],
		}
		h := openLoop(t, startClock(), loopOptions{
			amountOfWorkers:  2,
			postingsPerBatch: 1,
		})
		for _, wordPartition := range wordPartitions {
			peer := peerAfter(wordPartition, 1)
			h.transfers.answers[peer.Hash] = acceptedAnswer
		}
		h.reach(
			peerAfter(wordPartitions[0], 1),
			peerAfter(wordPartitions[1], 1),
			peerAfter(wordPartitions[2], 1),
		)
		answers := make(chan struct{})
		h.transfers.onOffer = func(yacymodel.Seed) { <-answers }
		for _, wordPartition := range wordPartitions {
			h.admit(t, wordPartition, urlsOf(1)...)
		}

		h.start(t)

		if len(h.transfers.offered) != 2 {
			t.Fatalf("offered = %v, want two batches in flight at once", h.transfers.offered)
		}
		close(answers)
		synctest.Wait()

		assertEachWordPartitionOfferedOnce(t, h.transfers.offered, wordPartitions)
	})
}

func openTwoBatches(t *testing.T) (*loopHarness, []yacymodel.WordPartition) {
	t.Helper()

	firstWordPartitions := wordPartitionsIn(1, postingsPerBatch)
	secondWordPartitions := wordPartitionsIn(40, postingsPerBatch)
	firstPeer := peerAfter(firstWordPartitions[len(firstWordPartitions)-1], 1)
	secondPeer := peerAfter(secondWordPartitions[len(secondWordPartitions)-1], 1)
	h := openLoop(t, startClock(), loopOptions{amountOfWorkers: 2})
	h.reach(firstPeer, secondPeer)
	h.transfers.answers[firstPeer.Hash] = acceptedAnswer
	h.transfers.answers[secondPeer.Hash] = acceptedAnswer
	wordPartitions := slices.Concat(firstWordPartitions, secondWordPartitions)
	for _, wordPartition := range wordPartitions {
		h.admit(t, wordPartition, urlsOf(1)...)
	}
	h.scheduleAdmissions(t)

	return h, wordPartitions
}

func assertEachWordPartitionOfferedOnce(
	t *testing.T,
	offered []offeredCall,
	wordPartitions []yacymodel.WordPartition,
) {
	t.Helper()

	offersPerWord := offersPerWordIn(offered)
	for _, wordPartition := range wordPartitions {
		if offersPerWord[wordPartition.Word] != 1 {
			t.Fatalf("offers per word = %v, want every word partition offered once", offersPerWord)
		}
	}
}

func offersPerWordIn(offered []offeredCall) map[yacymodel.Hash]int {
	offersPerWord := make(map[yacymodel.Hash]int)
	for _, call := range offered {
		for _, posting := range call.Postings {
			offersPerWord[posting.WordHash]++
		}
	}

	return offersPerWord
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

func TestLoopTakesNoArcOverABatchStillInFlightFromBeforeARingChange(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		first, second := fixtureWordPartition(), wordPartitionsIn(40, 1)[0]
		firstPeer, secondPeer := peerAfter(first, 1), peerAfter(second, 1)
		h := openLoop(t, startClock(), loopOptions{amountOfWorkers: 2})
		h.reach(firstPeer)
		h.transfers.answers[firstPeer.Hash] = acceptedAnswer
		h.transfers.answers[secondPeer.Hash] = acceptedAnswer
		firstAnswer := make(chan struct{})
		h.transfers.onOffer = func(peer yacymodel.Seed) {
			if peer.Hash == firstPeer.Hash {
				<-firstAnswer
			}
		}
		h.admit(t, first, urlsOf(1)...)
		h.start(t)
		h.admit(t, second, urlsOf(1)...)

		h.reach(firstPeer, secondPeer)
		h.settle()

		if got := len(h.transfers.offeredTo(secondPeer)); got != 0 {
			t.Fatalf(
				"transfers to the new peer = %d, want none while the older batch is in flight",
				got,
			)
		}
		close(firstAnswer)
		h.settle()
		if got := len(h.transfers.offeredTo(secondPeer)); got != 1 {
			t.Fatalf("transfers to the new peer = %d, want 1 once the older batch settled", got)
		}
	})
}
