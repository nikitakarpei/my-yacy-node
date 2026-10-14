package distributionloop_test

import (
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestLoopOffersANewPostingAtOnceAndNextWhenItsRefreshIsDue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		clk := startClock()
		sentAt := clk.Now()
		h := openLoop(t, clk, loopOptions{})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.start(t)

		h.admit(t, wordPartition, urlsOf(1)...)
		h.settle()

		if got := urlsIn(h.transfers.offeredTo(peer)); !slices.Equal(got, urlsOf(1)) {
			t.Fatalf("peer got %v, want the new posting at once", got)
		}
		if got, found := h.earliestDueAt(t); !found || !got.Equal(sentAt.Add(refreshInterval)) {
			t.Fatalf("next due time = %v, %v, want %v", got, found, sentAt.Add(refreshInterval))
		}
	})
}

func TestLoopSendsARefreshOfTheWholeWordPartitionToEveryWindowPeer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peers := []yacymodel.Seed{peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)}
		urls := urlsOf(3)
		h := openLoop(t, startClock(), loopOptions{redundancy: 3})
		h.reach(peers...)
		for _, peer := range peers {
			h.transfers.answers[peer.Hash] = acceptedAnswer
		}
		h.admit(t, wordPartition, urls[:2]...)
		h.start(t)
		h.admit(t, wordPartition, urls[2])
		h.settle()

		h.advance(refreshInterval)

		for _, peer := range peers {
			want := slices.Concat(urls[:2], urls[2:], sortedURLs(urls))
			if got := urlsIn(h.transfers.offeredTo(peer)); len(got) != len(want) {
				t.Fatalf("%v got %v, want the new postings and then the whole word partition",
					peer.Hash, got)
			}
			refresh := h.transfers.offeredTo(peer)[len(h.transfers.offeredTo(peer))-1]
			if got := urlsIn([]offeredCall{refresh}); !slices.Equal(got, sortedURLs(urls)) {
				t.Fatalf("%v got %v in the refresh, want %v", peer.Hash, got, sortedURLs(urls))
			}
		}
	})
}

func TestLoopReadsAnotherArcWhileAWindowPeerIsBusy(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		busyWordPartition, openWordPartition := wordPartitionsIn(1, 1)[0], wordPartitionsIn(40, 1)[0]
		busyPeer, openPeer := peerAfter(busyWordPartition, 1), peerAfter(openWordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{self: thisNodeFartherThan(busyWordPartition)})
		h.reach(busyPeer, openPeer)
		h.transfers.answers[busyPeer.Hash] = acceptedAnswer
		h.transfers.answers[openPeer.Hash] = acceptedAnswer
		h.eligibility.SendStarted(busyPeer.Hash)
		h.eligibility.SendEnded(busyPeer.Hash, 2*workInFlight)
		h.eligibility.SendStarted(busyPeer.Hash)
		h.admit(t, busyWordPartition, urlsOf(1)...)
		h.admit(t, openWordPartition, urlsOf(1)...)

		h.start(t)

		if got := len(h.transfers.offeredTo(busyPeer)); got != 0 {
			t.Fatalf("transfers to the busy peer = %d, want none", got)
		}
		if got := len(h.transfers.offeredTo(openPeer)); got != 1 {
			t.Fatalf("transfers to the open arc's peer = %d, want 1", got)
		}

		h.eligibility.SendEnded(busyPeer.Hash, 2*workInFlight)
		h.loop.ObserveReachablePeers(2)
		h.settle()

		if got := len(h.transfers.offeredTo(busyPeer)); got != 1 {
			t.Fatalf("transfers to the freed peer = %d, want 1", got)
		}
	})
}

func TestLoopReadsAnArcWhenThePauseOfItsWindowPeerEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.eligibility.OfferDeclined(peer.Hash, time.Hour)
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)
		if got := len(h.transfers.offeredTo(peer)); got != 0 {
			t.Fatalf("transfers to the paused peer = %d, want none", got)
		}

		h.advance(time.Hour - time.Second)
		if got := len(h.transfers.offeredTo(peer)); got != 0 {
			t.Fatalf("transfers before the pause ended = %d, want none", got)
		}
		h.advance(time.Second)

		if got := len(h.transfers.offeredTo(peer)); got != 1 {
			t.Fatalf("transfers once the pause ended = %d, want 1", got)
		}
	})
}

func TestLoopReadsADrainedArcAtItsEarliestDueTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urlsOf(1)...)
		h.scheduleAdmissions(t)
		h.recordHolders(t, wordPartition, peer.Hash)
		h.start(t)

		h.advance(refreshInterval - time.Second)
		if got := len(h.transfers.offeredTo(peer)); got != 0 {
			t.Fatalf("transfers before the refresh time = %d, want none", got)
		}
		h.advance(time.Second)

		if got := len(h.transfers.offeredTo(peer)); got != 1 {
			t.Fatalf("transfers at the refresh time = %d, want 1", got)
		}
	})
}

func TestLoopReadsADrainedArcAtOnceWhenAWordPartitionBecomesDue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartitions := wordPartitionsIn(sectorOf(fixtureWordPartition()), 2)
		peer := peerAfter(wordPartitions[1], 1)
		h := openLoop(t, startClock(), loopOptions{self: thisNodeFartherThan(wordPartitions[0])})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.recordHolders(t, wordPartitions[0], peer.Hash)
		h.start(t)

		h.admit(t, wordPartitions[1], urlsOf(1)...)
		h.settle()

		if got := len(h.transfers.offeredTo(peer)); got != 1 {
			t.Fatalf("transfers after the signal = %d, want 1 without waiting", got)
		}
	})
}

func TestLoopKeepsAnArcReadableWhenAPostingIsAdmittedWhileItIsRead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		clk := startClock()
		wordPartitions := wordPartitionsIn(1, 2)
		peer := peerAfter(wordPartitions[1], 1)
		h := openLoop(t, clk, loopOptions{self: thisNodeFartherThan(wordPartitions[0])})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.recordHolders(t, wordPartitions[0], peer.Hash)
		clk.advance(refreshInterval)
		exclusionAsks := 0
		h.offered.onExcludedAsked = func() {
			exclusionAsks++
			if exclusionAsks == 2 {
				h.eligibility.OfferDeclined(peer.Hash, time.Hour)
				h.admit(t, wordPartitions[1], urlsOf(1)...)
			}
		}

		h.start(t)
		if got := len(h.transfers.offeredTo(peer)); got != 0 {
			t.Fatalf("transfers while the peer is paused = %d, want none", got)
		}
		h.advance(time.Hour)

		offered := h.transfers.offeredTo(peer)
		if len(offered) != 1 || len(offered[0].Postings) != 1 {
			t.Fatalf(
				"offered = %d calls, want one transfer of the posting admitted during the read",
				len(offered),
			)
		}
	})
}

func TestLoopOffersAPostingAdmittedWhileItsWordPartitionIsSent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		urls := urlsOf(2)
		h.admit(t, wordPartition, urls[0])
		sends := 0
		h.transfers.onOffer = func(yacymodel.Seed) {
			sends++
			if sends == 1 {
				h.admit(t, wordPartition, urls[1])
			}
		}

		h.start(t)

		if got := urlsIn(h.transfers.offeredTo(peer)); !slices.Equal(got, sortedURLs(urls)) {
			t.Fatalf("peer got %v, want the posting admitted during the send too", got)
		}
	})
}

func TestLoopReadsAWordPartitionMadeDueBehindTheCursorOfItsArc(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartitions := wordPartitionsIn(sectorOf(fixtureWordPartition()), 4)
		peer := peerAfter(wordPartitions[3], 1)
		h := openLoop(t, startClock(), loopOptions{
			self:             thisNodeFartherThan(wordPartitions[0]),
			postingsPerBatch: 1,
		})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		for _, wordPartition := range wordPartitions[1:] {
			h.admit(t, wordPartition, urlsOf(1)...)
		}
		sends := 0
		h.transfers.onOffer = func(yacymodel.Seed) {
			sends++
			if sends == 2 {
				h.write(t, func(tx *vault.Txn) error {
					return h.schedule.Forget(tx, wordPartitions[3])
				})
				h.admit(t, wordPartitions[0], urlsOf(1)...)
			}
		}

		h.start(t)

		if got := urlsIn(h.transfers.offeredTo(peer)); len(got) != 3 {
			t.Fatalf(
				"peer got %v, want three sends including the word partition behind the cursor",
				got,
			)
		}
	})
}
