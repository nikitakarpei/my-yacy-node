package distributionloop_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributionloop"
)

const gaugeObservationWait = time.Minute

func assertOneArcIn(t *testing.T, h *loopHarness, want distributionloop.ArcState) {
	t.Helper()

	for _, state := range []distributionloop.ArcState{
		distributionloop.ArcInBatch,
		distributionloop.ArcPeerFailed,
		distributionloop.ArcPeerDeclined,
		distributionloop.ArcPeerBusy,
		distributionloop.ArcDrained,
		distributionloop.ArcWaitingForWorker,
	} {
		wantAmount := 0
		if state == want {
			wantAmount = 1
		}
		if got := h.observer.arcsIn(state); got != wantAmount {
			t.Fatalf("arcs in %s = %d, want %d", state, got, wantAmount)
		}
	}
}

func TestLoopCountsAnArcWhoseWindowPeerFailedAsPeerFailed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openLoop(
			t,
			startClock(),
			loopOptions{leastPause: time.Hour, unacceptedOffersBeforeExclusion: 2},
		)
		h.reach(peer)
		h.eligibility.OfferFailed(context.Background(), peer.Hash)
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)

		h.advance(gaugeObservationWait)

		assertOneArcIn(t, h, distributionloop.ArcPeerFailed)
	})
}

func TestLoopCountsAnArcWhoseWindowPeerDeclinedAsPeerDeclined(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(peer)
		h.eligibility.OfferDeclined(context.Background(), peer.Hash, time.Hour)
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)

		h.advance(gaugeObservationWait)

		assertOneArcIn(t, h, distributionloop.ArcPeerDeclined)
	})
}

func TestLoopCountsAnArcWhoseWindowPeerIsBusyAsPeerBusy(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(peer)
		h.eligibility.SendStarted(peer.Hash)
		h.eligibility.SendEnded(peer.Hash, 2*workInFlight)
		h.eligibility.SendStarted(peer.Hash)
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)

		h.advance(gaugeObservationWait)

		assertOneArcIn(t, h, distributionloop.ArcPeerBusy)
	})
}

func TestLoopCountsAnArcWithNothingDueAsDrained(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(peer)
		h.admit(t, wordPartition, urlsOf(1)...)
		h.recordHolders(t, wordPartition, peer.Hash)
		h.start(t)

		h.advance(gaugeObservationWait)

		assertOneArcIn(t, h, distributionloop.ArcDrained)
	})
}

func TestLoopCountsAnArcWhoseWorkerHoldsABatchAsInBatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		sending := make(chan struct{})
		h.transfers.onOffer = func(sentTo yacymodel.Hash) { <-sending }
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)

		h.advance(gaugeObservationWait)

		assertOneArcIn(t, h, distributionloop.ArcInBatch)
		close(sending)
	})
}
