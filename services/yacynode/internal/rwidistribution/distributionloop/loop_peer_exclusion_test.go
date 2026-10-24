package distributionloop_test

import (
	"testing"
	"testing/synctest"
)

func TestLoopExcludesAPeerWhoseOfferFailedAndOffersTheNextPeerAtTheRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		failingPeer, nextPeer := peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)
		h := openLoop(t, startClock(), loopOptions{unacceptedOffersBeforeExclusion: 1})
		h.reach(failingPeer, nextPeer)
		h.transfers.answers[failingPeer.Hash] = failedAnswer
		h.transfers.answers[nextPeer.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)

		h.advance(retryDelay)

		if got := len(h.transfers.offeredTo(failingPeer)); got != 1 {
			t.Fatalf("transfers to the failing peer = %d, want 1", got)
		}
		if got := len(h.transfers.offeredTo(nextPeer)); got != 1 {
			t.Fatalf("transfers to the next peer = %d, want 1 at the retry", got)
		}
		if got := h.amountOfUndeliveredPostings(t); got != 0 {
			t.Fatalf("undelivered postings = %d, want none", got)
		}
		if got := h.observer.peersExcluded; got != 1 {
			t.Fatalf("peers excluded = %d, want 1", got)
		}
	})
}

func TestLoopExcludesAPeerWhoseOfferWasDeclinedAndOffersTheNextPeerAtTheRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		decliningPeer, nextPeer := peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)
		h := openLoop(t, startClock(), loopOptions{unacceptedOffersBeforeExclusion: 1})
		h.reach(decliningPeer, nextPeer)
		h.transfers.answers[decliningPeer.Hash] = declinedAnswer
		h.transfers.answers[nextPeer.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)

		h.advance(retryDelay)

		if got := len(h.transfers.offeredTo(decliningPeer)); got != 1 {
			t.Fatalf("transfers to the declining peer = %d, want 1", got)
		}
		if got := len(h.transfers.offeredTo(nextPeer)); got != 1 {
			t.Fatalf("transfers to the next peer = %d, want 1 at the retry", got)
		}
		if got := h.amountOfUndeliveredPostings(t); got != 0 {
			t.Fatalf("undelivered postings = %d, want none", got)
		}
		if got := h.observer.peersExcluded; got != 1 {
			t.Fatalf("peers excluded = %d, want 1", got)
		}
	})
}

func TestLoopKeepsADecliningPeerResponsibleBelowTheExclusionThreshold(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		decliningPeer, nextPeer := peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)
		h := openLoop(t, startClock(), loopOptions{unacceptedOffersBeforeExclusion: 3})
		h.reach(decliningPeer, nextPeer)
		h.transfers.answers[decliningPeer.Hash] = declinedAnswer
		h.transfers.answers[nextPeer.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)

		h.advance(retryDelay)

		if got := len(h.transfers.offeredTo(decliningPeer)); got != 2 {
			t.Fatalf("transfers to the declining peer = %d, want 2", got)
		}
		if got := h.transfers.offeredTo(nextPeer); len(got) != 0 {
			t.Fatalf("transfers to the next peer = %v, want none", got)
		}
		if got := h.observer.peersExcluded; got != 0 {
			t.Fatalf("peers excluded = %d, want 0", got)
		}
	})
}
