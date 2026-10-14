package distributionloop_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestLoopOffersPostingsADecliningPeerLacksOnlyAtEachDoublingRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		decliningPeer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(decliningPeer)
		h.transfers.answers[decliningPeer.Hash] = declinedAnswer
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)

		for transfers, delay := range []time.Duration{retryDelay, 2 * retryDelay} {
			h.advance(delay - time.Second)
			if got := len(h.transfers.offeredTo(decliningPeer)); got != transfers+1 {
				t.Fatalf("transfers before retry %d = %d, want %d", transfers+1, got, transfers+1)
			}
			h.advance(time.Second)
			if got := len(h.transfers.offeredTo(decliningPeer)); got != transfers+2 {
				t.Fatalf("transfers at retry %d = %d, want %d", transfers+1, got, transfers+2)
			}
		}
	})
}

func TestLoopSendsAHolderNothingAgainWhileItsWordPartitionIsInRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		holder, decliningPeer := peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)
		h := openLoop(t, startClock(), loopOptions{redundancy: 3})
		h.reach(holder, decliningPeer)
		h.transfers.answers[holder.Hash] = acceptedAnswer
		h.transfers.answers[decliningPeer.Hash] = declinedAnswer
		h.admit(t, wordPartition, urlsOf(2)...)
		h.start(t)

		for range 3 {
			h.advance(rescanDelay)
		}

		if got := len(h.transfers.offeredTo(holder)); got != 1 {
			t.Fatalf("transfers to the holder = %d, want only the first before the retry", got)
		}
		if got := h.amountOfUndeliveredPostings(t); got != 2 {
			t.Fatalf("undelivered postings = %d, want 2 waiting for the retry", got)
		}
	})
}

func TestLoopOffersANewPostingOfAWordPartitionAtRedundancyOnTheNextPass(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		holder := peerAfter(wordPartition, 1)
		urls := urlsOf(2)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(holder)
		h.transfers.answers[holder.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urls[0])
		h.start(t)

		h.admit(t, wordPartition, urls[1])
		h.advance(rescanDelay)

		if got := len(h.transfers.offeredTo(holder)); got != 2 {
			t.Fatalf("transfers to the holder = %d, want the new posting on the next pass", got)
		}
		if got := h.amountOfUndeliveredPostings(t); got != 0 {
			t.Fatalf("undelivered postings = %d, want none", got)
		}
	})
}

func TestLoopOffersNewPostingsAtOnceAgainAfterARetryReachedRedundancy(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		urls := urlsOf(2)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = declinedAnswer
		h.admit(t, wordPartition, urls[0])
		h.start(t)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.advance(retryDelay)

		h.admit(t, wordPartition, urls[1])
		h.advance(rescanDelay)

		if got := len(h.transfers.offeredTo(peer)); got != 3 {
			t.Fatalf("transfers = %d, want the decline, the retry and the new posting", got)
		}
		if got := h.holdersOf(t, wordPartition); len(got) != 1 || got[0] != peer.Hash {
			t.Fatalf("holders = %v, want %v", got, []yacymodel.Hash{peer.Hash})
		}
		if got := h.amountOfUndeliveredPostings(t); got != 0 {
			t.Fatalf("undelivered postings = %d, want none", got)
		}
	})
}

func TestLoopPutsAWordPartitionInRetryWhenAHolderDeclinesItsNewPostings(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		holder := peerAfter(wordPartition, 1)
		urls := urlsOf(2)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(holder)
		h.transfers.answers[holder.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urls[0])
		h.start(t)
		h.transfers.answers[holder.Hash] = declinedAnswer
		h.admit(t, wordPartition, urls[1])
		h.advance(rescanDelay)

		h.advance(retryDelay - time.Second)
		if got := len(h.transfers.offeredTo(holder)); got != 2 {
			t.Fatalf("transfers before the retry = %d, want the delivery and one decline", got)
		}
		h.advance(time.Second)

		if got := len(h.transfers.offeredTo(holder)); got != 3 {
			t.Fatalf("transfers at the retry = %d, want the retry sent", got)
		}
	})
}
