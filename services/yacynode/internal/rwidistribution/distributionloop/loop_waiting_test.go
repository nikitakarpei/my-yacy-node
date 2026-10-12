package distributionloop_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
)

var waitingForCooldown = skipKind{
	offerKind:  string(wordpartitionoffer.UndeliveredOffer),
	skipReason: string(wordpartitionoffer.SkipRecipientInCooldown),
}

func TestLoopCountsAWordPartitionWaitingOnSeveralPassesOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		h := openWaitingLoop(t)
		h.putInCooldown(peerAfter(wordPartition, 1))
		h.admit(t, wordPartition, urlsOf(1)...)

		h.start(t)
		h.advance(sectorRescanDelay)
		h.advance(sectorRescanDelay)

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

		h.advance(refreshInterval + sectorRescanDelay)
		h.advance(sectorRescanDelay)
		h.advance(sectorRescanDelay)

		refreshWaitingForCooldown := skipKind{
			offerKind:  string(wordpartitionoffer.RefreshOffer),
			skipReason: string(wordpartitionoffer.SkipRecipientInCooldown),
		}
		if got := h.observer.wordPartitionsWaiting[refreshWaitingForCooldown]; got != 1 {
			t.Fatalf("word partitions waiting = %d, want 1 after three passes", got)
		}
	})
}

func TestLoopReplacesTheWaitingWordPartitionsOfASectorAtItsNextPassEnd(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openWaitingLoop(t)
		h.putInCooldown(peer)
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)

		h.eligibility.OfferAccepted(peer.Hash)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.advance(sectorRescanDelay)

		if got, found := h.observer.wordPartitionsWaiting[waitingForCooldown]; !found || got != 0 {
			t.Fatalf("word partitions waiting = %d (reported %t), want 0 once offered", got, found)
		}
	})
}

func TestLoopAddsUpTheWaitingWordPartitionsOfAllSectors(t *testing.T) {
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
			t.Fatalf("word partitions waiting = %d, want one of each sector", got)
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

func openWaitingLoop(t *testing.T) *loopHarness {
	t.Helper()

	return openLoop(t, startClock(), loopOptions{cooldown: time.Hour})
}

func (h *loopHarness) putInCooldown(peer yacymodel.Seed) {
	h.reach(append(h.roster.reachable, peer)...)
	h.eligibility.OfferDeclined(peer.Hash, 0)
}
