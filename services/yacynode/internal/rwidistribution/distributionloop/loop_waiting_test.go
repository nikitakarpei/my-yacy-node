package distributionloop_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
)

var waitingForCooldown = string(wordpartitionoffer.WaitRecipientInCooldown)

func TestLoopCountsAWordPartitionWaitingOnSeveralPassesOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		h := openWaitingLoop(t)
		h.putInCooldown(peerAfter(wordPartition, 1))
		h.admit(t, wordPartition, urlsOf(1)...)

		h.start(t)
		h.advance(rescanDelay)
		h.advance(rescanDelay)

		if got := h.observer.wordPartitionsWaiting[waitingForCooldown]; got != 1 {
			t.Fatalf("word partitions waiting = %d, want 1 after three passes", got)
		}
	})
}

func TestLoopCountsADueRefreshWaitingOnSeveralPassesOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		holder := peerAfter(wordPartition, 1)
		h := openWaitingLoop(t)
		h.reach(holder)
		h.transfers.answers[holder.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)
		h.eligibility.OfferDeclined(holder.Hash, 2*refreshInterval)

		h.advance(refreshInterval + rescanDelay)
		h.advance(rescanDelay)
		h.advance(rescanDelay)

		if got := h.observer.wordPartitionsWaiting[waitingForCooldown]; got != 1 {
			t.Fatalf("word partitions waiting = %d, want 1 after three passes", got)
		}
	})
}

func TestLoopReplacesTheWaitingWordPartitionsAtTheNextPassEnd(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openWaitingLoop(t)
		h.putInCooldown(peer)
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)

		h.eligibility.OfferAccepted(peer.Hash)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.advance(rescanDelay)

		if got, found := h.observer.wordPartitionsWaiting[waitingForCooldown]; !found || got != 0 {
			t.Fatalf("word partitions waiting = %d (reported %t), want 0 once offered", got, found)
		}
	})
}

func TestLoopCountsTheWaitingWordPartitionsAllAroundTheRing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartitions := []yacymodel.WordPartition{
			wordPartitionsIn(3, 1)[0],
			wordPartitionsIn(5, 1)[0],
		}
		h := openWaitingLoop(t)
		for _, wordPartition := range wordPartitions {
			h.putInCooldown(peerAfter(wordPartition, 1))
			h.admit(t, wordPartition, urlsOf(1)...)
		}

		h.start(t)

		if got := h.observer.wordPartitionsWaiting[waitingForCooldown]; got != 2 {
			t.Fatalf("word partitions waiting = %d, want one from each region of the ring", got)
		}
	})
}

func TestLoopAddsUpTheWaitingWordPartitionsOfBothWalks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartitions := wordPartitionsIn(3, 2)
		refreshed, undelivered := wordPartitions[0], wordPartitions[1]
		holder := peerAfter(refreshed, 1)
		h := openWaitingLoop(t)
		h.reach(holder)
		h.transfers.answers[holder.Hash] = acceptedAnswer
		h.admit(t, refreshed, urlsOf(1)...)
		h.start(t)
		h.eligibility.OfferDeclined(holder.Hash, 2*refreshInterval)
		peerInCooldown := peerAfter(undelivered, 1)
		h.reach(holder, peerInCooldown)
		h.eligibility.OfferDeclined(peerInCooldown.Hash, 2*refreshInterval)
		h.admit(t, undelivered, urlsOf(1)...)

		h.advance(refreshInterval + rescanDelay)

		if got := h.observer.wordPartitionsWaiting[waitingForCooldown]; got != 2 {
			t.Fatalf("word partitions waiting = %d, want one from each walk", got)
		}
	})
}

func TestLoopReportsNoWaitingWordPartitionsOfAnUnwrittenBatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartitions := wordPartitionsIn(3, 2)
		offeredPeer := peerAfter(wordPartitions[1], 1)
		h := openWaitingLoop(t)
		h.putInCooldown(peerAfter(wordPartitions[0], 1))
		h.reach(append(h.roster.reachable, offeredPeer)...)
		h.transfers.answers[offeredPeer.Hash] = acceptedAnswer
		h.transfers.onOffer = func(yacymodel.Seed) {
			if err := h.v.Close(); err != nil {
				t.Errorf("Close: %v", err)
			}
		}
		for _, wordPartition := range wordPartitions {
			h.admit(t, wordPartition, urlsOf(1)...)
		}

		h.start(t)

		if len(h.observer.wordPartitionsWaiting) != 0 {
			t.Fatalf(
				"word partitions waiting = %v, want none reported for an unwritten batch",
				h.observer.wordPartitionsWaiting,
			)
		}
	})
}

func TestLoopCountsAWordPartitionInRetryAsWaiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		decliningPeer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(decliningPeer)
		h.transfers.answers[decliningPeer.Hash] = declinedAnswer
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)

		h.advance(rescanDelay)

		if got := h.observer.wordPartitionsWaiting[string(
			wordpartitionoffer.WaitWordPartitionInRetry)]; got != 1 {
			t.Fatalf("word partitions in retry = %d, want 1", got)
		}
	})
}

func openWaitingLoop(t *testing.T) *loopHarness {
	t.Helper()

	return openLoop(t, startClock(), loopOptions{cooldown: time.Hour})
}

func (h *loopHarness) putInCooldown(peer yacymodel.Seed) {
	h.reach(append(h.roster.reachable, peer)...)
	h.eligibility.OfferDeclined(peer.Hash, 0)
}
