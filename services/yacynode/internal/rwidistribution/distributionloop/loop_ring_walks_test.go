package distributionloop_test

import (
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestLoopTakesTheUndeliveredPostingsInRingOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		later, earlier := wordPartitionsIn(40, 1)[0], wordPartitionsIn(3, 1)[0]
		h := openWalkLoop(t, 1)
		h.admit(t, later, urlsOf(1)...)
		h.admit(t, earlier, urlsOf(1)...)

		h.start(t)

		if len(h.transfers.offered) != 2 ||
			h.transfers.offered[0].Postings[0].WordHash != earlier.Word ||
			h.transfers.offered[1].Postings[0].WordHash != later.Word {
			t.Fatalf("offered %+v, want sector 3 before sector 40", h.transfers.offered)
		}
	})
}

func TestLoopOffersAgainAtOnceWhileAPassHoldsMoreThanABatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartitions := wordPartitionsIn(3, postingsPerBatch+1)
		h := openWalkLoop(t, postingsPerBatch)
		for _, wordPartition := range wordPartitions {
			h.admit(t, wordPartition, urlsOf(1)...)
		}

		h.start(t)

		if len(h.transfers.offered) != 2 || len(h.transfers.offered[1].Postings) != 1 {
			t.Fatalf(
				"offered %+v, want a full batch, then the one posting left",
				h.transfers.offered,
			)
		}
		assertEachWordPartitionOfferedOnce(t, h.transfers.offered, wordPartitions)
	})
}

func TestLoopReadsTheUndeliveredPostingsAgainOnlyAfterTheRescanDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartitions := wordPartitionsIn(3, 2)
		h := openWalkLoop(t, postingsPerBatch)
		h.admit(t, wordPartitions[0], urlsOf(1)...)
		h.start(t)

		h.admit(t, wordPartitions[1], urlsOf(1)...)
		h.advance(rescanDelay - time.Millisecond)
		if len(h.transfers.offered) != 1 {
			t.Fatalf(
				"offered %+v, want the postings unread before the rescan delay",
				h.transfers.offered,
			)
		}
		h.advance(time.Millisecond)

		if len(h.transfers.offered) != 2 ||
			h.transfers.offered[1].Postings[0].WordHash != wordPartitions[1].Word {
			t.Fatalf(
				"offered %+v, want the new posting after the rescan delay",
				h.transfers.offered,
			)
		}
	})
}

func TestLoopTakesTheNextBatchAtOnceAfterABatchThatEndedNoPass(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const postingsPerSmallBatch = 2
		wordPartitions := wordPartitionsIn(3, 10*postingsPerSmallBatch+1)
		lastWordPartition := wordPartitions[len(wordPartitions)-1]
		pausedPeer := peerAfter(wordPartitions[len(wordPartitions)-2], 1)
		peer := peerAfter(lastWordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{
			self:             thisNodeFartherThan(wordPartitions[0]),
			leastPause:       time.Hour,
			postingsPerBatch: postingsPerSmallBatch,
		})
		h.reach(pausedPeer, peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.eligibility.OfferDeclined(pausedPeer.Hash, 0)
		for _, wordPartition := range wordPartitions {
			h.admit(t, wordPartition, urlsOf(1)...)
		}

		h.start(t)

		if got := h.holdersOf(t, lastWordPartition); len(got) != 1 || got[0] != peer.Hash {
			t.Fatalf("holders = %v, want %v without waiting for the rescan delay", got, peer.Hash)
		}
	})
}

func TestLoopGivesEveryWorkerABatchFromOneBusyRegionOfTheRing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const amountOfWorkers = 8
		wordPartitions := wordPartitionsIn(3, amountOfWorkers)
		peer := peerAfter(wordPartitions[amountOfWorkers-1], 1)
		h := openLoop(t, startClock(), loopOptions{
			self:             thisNodeFartherThan(wordPartitions[0]),
			amountOfWorkers:  amountOfWorkers,
			postingsPerBatch: 1,
		})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		meeting := newCallMeeting(amountOfWorkers)
		h.transfers.onOffer = func(yacymodel.Seed) { meeting.arrive() }
		for _, wordPartition := range wordPartitions {
			h.admit(t, wordPartition, urlsOf(1)...)
		}

		h.start(t)

		meeting.assertMet(t)
		assertEachWordPartitionOfferedOnce(t, h.transfers.offered, wordPartitions)
	})
}

func TestLoopNeverOffersAWordPartitionInTwoBatchesAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		slowPeer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{amountOfWorkers: 2})
		h.reach(slowPeer)
		h.transfers.answers[slowPeer.Hash] = acceptedAnswer
		slowAnswer := make(chan struct{})
		h.transfers.onOffer = func(yacymodel.Seed) { <-slowAnswer }
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)

		h.admit(t, wordPartition, urlsOf(2)...)
		for range 3 {
			h.advance(rescanDelay)
		}

		if got := len(h.transfers.offeredTo(slowPeer)); got != 1 {
			t.Fatalf("transfers = %d, want one while the first batch is still out", got)
		}
		close(slowAnswer)
		synctest.Wait()
		h.advance(rescanDelay)

		if got := h.amountOfUndeliveredPostings(t); got != 0 {
			t.Fatalf("undelivered postings = %d, want none once the slow batch is done", got)
		}
	})
}

func TestLoopAlternatesBetweenUndeliveredPostingsAndDueRefreshes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartitions := wordPartitionsIn(3, 6)
		refreshed, undelivered := wordPartitions[:3], wordPartitions[3:]
		h := openWalkLoop(t, 1)
		for _, wordPartition := range refreshed {
			h.admit(t, wordPartition, urlsOf(1)...)
		}
		h.start(t)
		h.transfers.offered = nil
		for _, wordPartition := range undelivered {
			h.admit(t, wordPartition, urlsOf(1)...)
		}

		h.advance(refreshInterval)

		if len(h.transfers.offered) != len(wordPartitions) {
			t.Fatalf("offered %+v, want every word partition once", h.transfers.offered)
		}
		for index := 1; index < len(h.transfers.offered); index++ {
			if isRefreshOf(h.transfers.offered[index], refreshed) ==
				isRefreshOf(h.transfers.offered[index-1], refreshed) {
				t.Fatalf("batches %d and %d come from one walk, want the walks to take turns",
					index-1, index)
			}
		}
	})
}

func isRefreshOf(call offeredCall, refreshed []yacymodel.WordPartition) bool {
	return slices.ContainsFunc(refreshed, func(wordPartition yacymodel.WordPartition) bool {
		return wordPartition.Word == call.Postings[0].WordHash
	})
}

func TestLoopGivesEveryBatchToTheWalkThatHasWork(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		refreshed := wordPartitionsIn(3, 3)
		h := openWalkLoop(t, 1)
		for _, wordPartition := range refreshed {
			h.admit(t, wordPartition, urlsOf(1)...)
		}
		h.start(t)
		h.transfers.offered = nil

		h.advance(refreshInterval)

		assertEachWordPartitionOfferedOnce(t, h.transfers.offered, refreshed)
	})
}

func TestLoopRefreshesTheWordPartitionsBehindABlockedRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartitions := wordPartitionsIn(3, 2)
		blocked, behind := wordPartitions[0], wordPartitions[1]
		blockedHolder, holder := peerAfter(blocked, 1), peerAfter(behind, 1)
		h := openLoop(t, startClock(), loopOptions{
			self:             thisNodeFartherThan(blocked),
			postingsPerBatch: 1,
		})
		h.reach(blockedHolder, holder)
		h.transfers.answers[blockedHolder.Hash] = acceptedAnswer
		h.transfers.answers[holder.Hash] = acceptedAnswer
		h.admit(t, blocked, urlsOf(1)...)
		h.start(t)
		h.advance(time.Minute)
		h.admit(t, behind, urlsOf(1)...)
		h.advance(rescanDelay)
		h.eligibility.OfferDeclined(blockedHolder.Hash, 2*refreshInterval)

		h.advance(refreshInterval)

		if got := len(h.transfers.offeredTo(holder)); got != 2 {
			t.Fatalf("transfers to the holder behind = %d, want its refresh too", got)
		}
		if got := len(h.transfers.offeredTo(blockedHolder)); got != 1 {
			t.Fatalf("transfers to the blocked holder = %d, want no refresh yet", got)
		}
	})
}

func openWalkLoop(t *testing.T, postingsPerWalkBatch int) *loopHarness {
	t.Helper()

	h := openLoop(t, startClock(), loopOptions{
		redundancy:       2,
		postingsPerBatch: postingsPerWalkBatch,
	})
	peer := seed(yacymodel.WordHash("peer"))
	h.reach(peer)
	h.transfers.answers[peer.Hash] = acceptedAnswer

	return h
}
