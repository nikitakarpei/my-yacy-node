package distributioncycle_test

import (
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const callMeetingWait = time.Second

func TestCycleOffersABatchToAllItsPeersAtTheSameTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		h := openCycle(t, startClock(), cycleOptions{redundancy: 3})
		h.reach(peerAfter(wordPartition, 1), peerAfter(wordPartition, 2))
		meeting := newCallMeeting(2)
		h.transfers.onOffer = meeting.arrive
		h.admit(t, wordPartition, urlsOf(1)...)

		h.runCycle(t)

		meeting.assertMet(t)
	})
}

func TestCycleOffersBatchesOfDifferentSectorsAtTheSameTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, wordPartitions := openTwoSectors(t)
		meeting := newCallMeeting(2)
		h.transfers.onOffer = meeting.arrive

		h.runCycle(t)

		meeting.assertMet(t)
		assertEachWordPartitionOfferedOnce(t, h.transfers.offered, wordPartitions)
	})
}

func TestCycleReadsTheDuePostingsOfBatchesAtTheSameTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := openTwoSectors(t)
		meeting := newCallMeeting(2)
		h.index.onRead = meeting.arrive

		h.runCycle(t)

		meeting.assertMet(t)
	})
}

func openTwoSectors(t *testing.T) (*cycleHarness, []yacymodel.WordPartition) {
	t.Helper()

	wordPartitions := slices.Concat(
		wordPartitionsIn(1, postingsPerBatch),
		wordPartitionsIn(40, postingsPerBatch),
	)
	peer := peerAfter(fixtureWordPartition(), 1)
	h := openCycle(t, startClock(), cycleOptions{redundancy: 2, batchConcurrency: 2})
	h.reach(peer)
	h.transfers.answers[peer.Hash] = acceptedAnswer
	for _, wordPartition := range wordPartitions {
		h.admit(t, wordPartition, urlsOf(1)...)
	}

	return h, wordPartitions
}

func assertEachWordPartitionOfferedOnce(
	t *testing.T,
	offered []offeredCall,
	wordPartitions []yacymodel.WordPartition,
) {
	t.Helper()

	offersPerWord := make(map[yacymodel.Hash]int)
	for _, call := range offered {
		for _, posting := range call.Postings {
			offersPerWord[posting.WordHash]++
		}
	}
	for _, wordPartition := range wordPartitions {
		if offersPerWord[wordPartition.Word] != 1 {
			t.Fatalf("offers per word = %v, want every word partition offered once", offersPerWord)
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
