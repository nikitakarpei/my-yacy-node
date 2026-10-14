package distributionloop_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
)

var waitingForPause = string(wordpartitionoffer.WaitRecipientPaused)

func TestLoopCountsAWordPartitionWaitingOnSeveralPassesOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		h := openWaitingLoop(t)
		h.pause(peerAfter(wordPartition, 1))
		h.admit(t, wordPartition, urlsOf(1)...)

		h.start(t)
		h.advance(rescanDelay)
		h.advance(rescanDelay)

		if got := h.observer.wordPartitionsWaiting[waitingForPause]; got != 1 {
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

		if got := h.observer.wordPartitionsWaiting[waitingForPause]; got != 1 {
			t.Fatalf("word partitions waiting = %d, want 1 after three passes", got)
		}
	})
}

func TestLoopReplacesTheWaitingWordPartitionsAtTheNextPassEnd(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openWaitingLoop(t)
		h.pause(peer)
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)

		h.eligibility.OfferAccepted(peer.Hash)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.advance(rescanDelay)

		if got, found := h.observer.wordPartitionsWaiting[waitingForPause]; !found || got != 0 {
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
			h.pause(peerAfter(wordPartition, 1))
			h.admit(t, wordPartition, urlsOf(1)...)
		}

		h.start(t)

		if got := h.observer.wordPartitionsWaiting[waitingForPause]; got != 2 {
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
		pausedPeer := peerAfter(undelivered, 1)
		h.reach(holder, pausedPeer)
		h.eligibility.OfferDeclined(pausedPeer.Hash, 2*refreshInterval)
		h.admit(t, undelivered, urlsOf(1)...)

		h.advance(refreshInterval + rescanDelay)

		if got := h.observer.wordPartitionsWaiting[waitingForPause]; got != 2 {
			t.Fatalf("word partitions waiting = %d, want one from each walk", got)
		}
	})
}

func TestLoopReportsNoWaitingWordPartitionsOfAnUnwrittenBatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartitions := wordPartitionsIn(3, 2)
		offeredPeer := peerAfter(wordPartitions[1], 1)
		h := openWaitingLoop(t)
		h.pause(peerAfter(wordPartitions[0], 1))
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

func TestLoopReportsAWaitingWordPartitionBeforeItsPassEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		waiting, offered := wordPartitionsIn(3, 1)[0], wordPartitionsIn(40, 1)[0]
		offeredPeer := peerAfter(offered, 1)
		h := openLoop(t, startClock(), loopOptions{leastPause: time.Hour, postingsPerBatch: 1})
		h.pause(peerAfter(waiting, 1))
		h.reach(append(h.roster.reachable, offeredPeer)...)
		h.transfers.answers[offeredPeer.Hash] = acceptedAnswer
		waitingAtTheOffer := h.waitingWhenOffered(offeredPeer)
		h.admit(t, waiting, urlsOf(1)...)
		h.admit(t, offered, urlsOf(1)...)

		h.start(t)

		if *waitingAtTheOffer != 1 {
			t.Fatalf("word partitions waiting = %d at the next batch, want 1", *waitingAtTheOffer)
		}
	})
}

func TestLoopReportsAWordPartitionNoLongerWaitingOnceItsSectorIsReadAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openWaitingLoop(t)
		h.pause(peer)
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)
		h.eligibility.OfferAccepted(peer.Hash)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		waitingAtTheOffer := h.waitingWhenOffered(peer)

		h.advance(rescanDelay)

		if *waitingAtTheOffer != 0 {
			t.Fatalf("word partitions waiting = %d at its offer, want 0", *waitingAtTheOffer)
		}
	})
}

func TestLoopReportsNoWaitingInASectorThatEmptiedOnceItIsReadAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		emptied, next, last := wordPartitionsIn(3, 1)[0], wordPartitionsIn(40, 1)[0],
			wordPartitionsIn(50, 1)[0]
		nextPeer, lastPeer := peerAfter(next, 1), peerAfter(last, 1)
		h := openLoop(t, startClock(), loopOptions{leastPause: time.Hour, postingsPerBatch: 1})
		h.pause(peerAfter(emptied, 1))
		h.admit(t, emptied, urlsOf(1)...)
		h.start(t)
		h.markDelivered(t, emptied)
		h.reach(append(h.roster.reachable, nextPeer, lastPeer)...)
		h.transfers.answers[nextPeer.Hash] = acceptedAnswer
		h.transfers.answers[lastPeer.Hash] = acceptedAnswer
		waitingAtTheOffer := h.waitingWhenOffered(nextPeer)
		h.admit(t, next, urlsOf(1)...)
		h.admit(t, last, urlsOf(1)...)

		h.advance(rescanDelay)

		if *waitingAtTheOffer != 0 {
			t.Fatalf("word partitions waiting = %d past the emptied sector, want 0",
				*waitingAtTheOffer)
		}
	})
}

func TestLoopDropsTheWaitingOfABatchFromAnEarlierPass(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartitions := wordPartitionsIn(3, 2)
		waiting, slow := wordPartitions[0], wordPartitions[1]
		slowPeer := peerAfter(slow, 1)
		h := openLoop(t, startClock(), loopOptions{
			leastPause:       time.Hour,
			amountOfWorkers:  2,
			postingsPerBatch: 2,
		})
		h.pause(peerAfter(waiting, 1))
		h.reach(append(h.roster.reachable, slowPeer)...)
		h.transfers.answers[slowPeer.Hash] = acceptedAnswer
		slowAnswer := make(chan struct{})
		h.transfers.onOffer = func(peer yacymodel.Seed) {
			if peer.Hash == slowPeer.Hash {
				<-slowAnswer
			}
		}
		h.admit(t, waiting, urlsOf(1)...)
		h.admit(t, slow, urlsOf(1)...)
		h.start(t)
		h.advance(rescanDelay)
		waitingBefore := h.observer.wordPartitionsWaitingFor(waitingForPause)

		close(slowAnswer)
		synctest.Wait()

		if got := h.observer.wordPartitionsWaitingFor(waitingForPause); got != waitingBefore {
			t.Fatalf("word partitions waiting = %d after the late batch, want %d unchanged",
				got, waitingBefore)
		}
	})
}

func (h *loopHarness) waitingWhenOffered(peer yacymodel.Seed) *int {
	waitingAtTheOffer := -1
	h.transfers.onOffer = func(offeredPeer yacymodel.Seed) {
		if offeredPeer.Hash == peer.Hash {
			waitingAtTheOffer = h.observer.wordPartitionsWaitingFor(waitingForPause)
		}
	}

	return &waitingAtTheOffer
}

func (h *loopHarness) markDelivered(t *testing.T, wordPartition yacymodel.WordPartition) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		postings, err := h.undelivered.PostingsOf(tx, wordPartition)
		if err != nil {
			return err
		}
		for _, posting := range postings {
			if _, err := h.undelivered.MarkDelivered(tx, posting); err != nil {
				return err
			}
		}

		return nil
	})
}

func openWaitingLoop(t *testing.T) *loopHarness {
	t.Helper()

	return openLoop(t, startClock(), loopOptions{leastPause: time.Hour})
}

func (h *loopHarness) pause(peer yacymodel.Seed) {
	h.reach(append(h.roster.reachable, peer)...)
	h.eligibility.OfferDeclined(peer.Hash, 0)
}
