package distributionloop_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
)

var waitingForBusyRecipient = string(wordpartitionoffer.WaitRecipientBusy)

type pacingHarness struct {
	*loopHarness

	peer          yacymodel.Seed
	sent, waiting yacymodel.WordPartition
	peerAnswer    chan struct{}
}

func openPacingLoop(t *testing.T, usualSendTime time.Duration) *pacingHarness {
	t.Helper()

	sent, waiting := wordPartitionsIn(1, 1)[0], wordPartitionsIn(40, 1)[0]
	h := &pacingHarness{
		loopHarness: openLoop(t, startClock(), loopOptions{
			self:             thisNodeFartherThan(sent),
			amountOfWorkers:  2,
			postingsPerBatch: 1,
		}),
		peer:       peerAfter(waiting, 1),
		sent:       sent,
		waiting:    waiting,
		peerAnswer: make(chan struct{}),
	}
	h.reach(h.peer)
	h.transfers.answers[h.peer.Hash] = acceptedAnswer
	h.transfers.onOffer = func(yacymodel.Seed) { <-h.peerAnswer }
	h.eligibility.SendStarted(h.peer.Hash)
	h.eligibility.SendEnded(h.peer.Hash, usualSendTime)

	return h
}

func (h *pacingHarness) offerTheSecondWordPartitionDuringTheFirstSend(t *testing.T) {
	t.Helper()

	h.admit(t, h.sent, urlsOf(1)...)
	h.start(t)
	h.admit(t, h.waiting, urlsOf(1)...)
	h.advance(rescanDelay)
}

func (h *pacingHarness) answer() {
	close(h.peerAnswer)
	synctest.Wait()
}

func TestLoopHoldsBackTheWordPartitionsOfASlowPeerWhileASendIsInFlight(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := openPacingLoop(t, 2*workInFlight)

		h.offerTheSecondWordPartitionDuringTheFirstSend(t)

		if got := len(h.transfers.offeredTo(h.peer)); got != 1 {
			t.Fatalf("transfers to the slow peer = %d, want 1 at a time", got)
		}
		if got := h.observer.wordPartitionsWaitingFor(waitingForBusyRecipient); got != 1 {
			t.Fatalf("word partitions waiting for a busy recipient = %d, want 1", got)
		}
		h.answer()
	})
}

func TestLoopSendsTheHeldBackWordPartitionOnceTheSlowPeerAnswers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := openPacingLoop(t, 2*workInFlight)
		h.offerTheSecondWordPartitionDuringTheFirstSend(t)

		h.answer()
		h.advance(rescanDelay)

		if got := len(h.transfers.offeredTo(h.peer)); got != 2 {
			t.Fatalf("transfers to the slow peer = %d, want 2", got)
		}
		if got := h.amountOfUndeliveredPostings(t); got != 0 {
			t.Fatalf("undelivered postings = %d, want none", got)
		}
	})
}

func TestLoopSendsSeveralWordPartitionsAtOnceToAFastPeer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := openPacingLoop(t, workInFlight/4)

		h.offerTheSecondWordPartitionDuringTheFirstSend(t)

		if got := len(h.transfers.offeredTo(h.peer)); got != 2 {
			t.Fatalf("transfers to the fast peer = %d, want 2 in flight at once", got)
		}
		h.answer()
	})
}

func TestLoopEndsTheSendToAPeerThatDoesNotAccept(t *testing.T) {
	for name, answer := range map[string]fakeAnswer{
		"declined": declinedAnswer,
		"failed":   failedAnswer,
	} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h := openPacingLoop(t, 2*workInFlight)
				h.transfers.answers[h.peer.Hash] = answer
				h.admit(t, h.sent, urlsOf(1)...)
				h.start(t)

				h.answer()

				if len(h.transfers.offeredTo(h.peer)) != 1 || h.eligibility.IsBusy(h.peer.Hash) {
					t.Fatal("peer is busy after its only send ended, want it free")
				}
			})
		})
	}
}

func TestLoopEndsTheSendCancelledWhileInFlight(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := openPacingLoop(t, 2*workInFlight)
		h.admit(t, h.sent, urlsOf(1)...)
		cancel := h.startCancellable(t)

		cancel()
		h.answer()

		if len(h.transfers.offeredTo(h.peer)) != 1 || h.eligibility.IsBusy(h.peer.Hash) {
			t.Fatal("peer is busy after its cancelled send ended, want it free")
		}
	})
}
