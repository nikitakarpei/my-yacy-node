package distributionloop_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributionloop"
)

func TestLoopRunReturnsWhenItsContextEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urlsOf(1)...)

		ctx, cancel := context.WithCancel(context.Background())
		h.transfers.onOffer = func(yacymodel.Seed) { cancel() }
		h.loop.Run(ctx)

		if len(h.transfers.offered) != 1 {
			t.Fatalf(
				"offered = %v, want a single offer before the context ended",
				h.transfers.offered,
			)
		}
	})
}

func TestLoopSkipsBatchesWhileTooFewPeersAreReachable(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{minReachablePeers: 2})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urlsOf(1)...)

		h.start(t)

		if len(h.transfers.offered) != 0 {
			t.Fatalf(
				"offered = %v, want no offers below the reachable-peer floor",
				h.transfers.offered,
			)
		}
		if h.observer.batchesSkipped[string(distributionloop.SkipTooFewReachablePeers)] != 1 {
			t.Fatalf("batchesSkipped = %v, want one skip below the reachable-peer floor",
				h.observer.batchesSkipped)
		}
		if got := h.amountOfUndeliveredPostings(t); got != 1 {
			t.Fatalf("undelivered postings = %d, want 1 left untouched by a skipped batch", got)
		}
	})
}

func TestLoopOffersAgainOnceEnoughPeersAreReachable(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer, farPeer := peerAfter(wordPartition, 1), peerAfter(wordPartition, 1<<40)
		h := openLoop(t, startClock(), loopOptions{minReachablePeers: 2})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urlsOf(1)...)
		h.start(t)

		h.reach(peer, farPeer)
		h.advance(rescanDelay)

		if got := urlsIn(h.transfers.offeredTo(peer)); !slices.Equal(got, urlsOf(1)) {
			t.Fatalf("peer got %v, want the posting once enough peers are reachable", got)
		}
		if got := h.amountOfUndeliveredPostings(t); got != 0 {
			t.Fatalf("undelivered postings = %d, want none", got)
		}
	})
}

func TestLoopReportsTheBacklogWhileItSkipsBatches(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		clk := startClock()
		delivered, undelivered := wordPartitionOf(0), wordPartitionOf(1)
		h := openLoop(t, clk, loopOptions{minReachablePeers: 2})
		h.admit(t, undelivered, urlsOf(3)...)
		h.recordHolders(t, delivered, peerAfter(delivered, 1).Hash)
		clk.advance(refreshInterval + time.Minute)

		h.start(t)

		if h.observer.undeliveredPostings != 3 || h.observer.longestRefreshLateness != time.Minute {
			t.Fatalf("undelivered postings = %d, longest refresh lateness = %v, want 3 and 1m",
				h.observer.undeliveredPostings, h.observer.longestRefreshLateness)
		}
	})
}

func TestLoopReportsTheBacklogAgainOnlyWhenItsTimerFires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		h := openLoop(t, startClock(), loopOptions{minReachablePeers: 2})
		h.admit(t, wordPartition, urlsOf(3)...)
		h.start(t)

		h.admit(t, wordPartitionOf(1), urlsOf(2)...)
		synctest.Wait()
		if h.observer.undeliveredPostings != 3 {
			t.Fatalf("undelivered postings = %d, want 3 until the timer fires",
				h.observer.undeliveredPostings)
		}
		h.advance(time.Minute)

		if h.observer.undeliveredPostings != 5 {
			t.Fatalf("undelivered postings = %d, want 5 after the timer fired",
				h.observer.undeliveredPostings)
		}
	})
}

func TestLoopReportsNoRefreshLatenessBeforeTheRefreshTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		delivered := wordPartitionOf(0)
		h := openLoop(t, startClock(), loopOptions{minReachablePeers: 2})
		h.recordHolders(t, delivered, peerAfter(delivered, 1).Hash)
		h.observer.longestRefreshLateness = time.Hour

		h.start(t)

		if h.observer.longestRefreshLateness != 0 {
			t.Fatalf("longest refresh lateness = %v, want 0", h.observer.longestRefreshLateness)
		}
	})
}

func TestLoopDeliversANewPostingToEveryResponsiblePeer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peers := []yacymodel.Seed{peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)}
		h := openLoop(t, startClock(), loopOptions{redundancy: 3})
		h.reach(peers...)
		for _, peer := range peers {
			h.transfers.answers[peer.Hash] = acceptedAnswer
		}
		h.admit(t, wordPartition, urlsOf(1)...)

		h.start(t)

		for _, peer := range peers {
			if got := urlsIn(h.transfers.offeredTo(peer)); !slices.Equal(got, urlsOf(1)) {
				t.Fatalf("%v got %v, want the new posting", peer.Hash, got)
			}
		}
		if got := h.amountOfUndeliveredPostings(t); got != 0 {
			t.Fatalf("undelivered postings = %d, want the new posting delivered", got)
		}
		want := []yacymodel.Hash{peers[0].Hash, peers[1].Hash}
		if got := h.holdersOf(t, wordPartition); !slices.Equal(got, want) {
			t.Fatalf("holders = %v, want %v", got, want)
		}
	})
}

func TestLoopSendsTheWholeWordPartitionToAMissingHolder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		holder, missingHolder := peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)
		urls := urlsOf(4)
		h := openLoop(t, startClock(), loopOptions{redundancy: 3})
		h.transfers.answers[holder.Hash] = acceptedAnswer
		h.transfers.answers[missingHolder.Hash] = acceptedAnswer
		h.reach(holder)
		h.admit(t, wordPartition, urls[:3]...)
		h.start(t)

		h.reach(holder, missingHolder)
		h.admit(t, wordPartition, urls[3])
		h.advance(rescanDelay)

		if got := urlsIn(h.transfers.offeredTo(holder)); !slices.Equal(got, sortedURLs(urls)) {
			t.Fatalf("holder got %v, want its first postings and then only the new one", got)
		}
		if got := urlsIn(
			h.transfers.offeredTo(missingHolder),
		); !slices.Equal(
			got,
			sortedURLs(urls),
		) {
			t.Fatalf(
				"missing holder got %v, want the whole word partition %v",
				got,
				sortedURLs(urls),
			)
		}
		if got := h.holdersOf(t, wordPartition); !slices.Contains(got, missingHolder.Hash) {
			t.Fatalf("holders = %v, want %v among them", got, missingHolder.Hash)
		}
		if got := h.amountOfUndeliveredPostings(t); got != 0 {
			t.Fatalf("undelivered postings = %d, want none", got)
		}
	})
}

func TestLoopLetsAWordPartitionWaitForAResponsiblePeerInCooldown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		holder, peerInCooldown := peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)
		h := openLoop(t, startClock(), loopOptions{redundancy: 3, cooldown: time.Hour})
		h.reach(holder, peerInCooldown)
		h.transfers.answers[holder.Hash] = acceptedAnswer
		h.transfers.answers[peerInCooldown.Hash] = declinedAnswer
		h.admit(t, wordPartition, urlsOf(2)...)
		h.start(t)

		h.advance(time.Minute)

		if len(h.transfers.offered) != 2 {
			t.Fatalf(
				"offered = %v, want only the first two: one peer is in cooldown",
				h.transfers.offered,
			)
		}
		if got := h.amountOfUndeliveredPostings(t); got != 2 {
			t.Fatalf("undelivered postings = %d, want 2 waiting for the peer in cooldown", got)
		}

		h.transfers.answers[peerInCooldown.Hash] = acceptedAnswer
		h.advance(time.Hour)

		if got := urlsIn(h.transfers.offeredTo(peerInCooldown)); len(got) != 4 {
			t.Fatalf("peer after its cooldown got %v, want the word partition again", got)
		}
		if got := h.amountOfUndeliveredPostings(t); got != 0 {
			t.Fatalf("undelivered postings = %d, want none once the cooldown ended", got)
		}
	})
}

func TestLoopRecordsNoHolderOnADeclinedOffer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = declinedAnswer
		h.admit(t, wordPartition, urlsOf(1)...)

		h.start(t)

		if holders := h.holdersOf(t, wordPartition); len(holders) != 0 {
			t.Fatalf("holders = %v, want none after a declined offer", holders)
		}
		if got := h.amountOfUndeliveredPostings(t); got != 1 {
			t.Fatalf("undelivered postings = %d, want 1", got)
		}
	})
}

func TestLoopRefreshesTheHoldersAndDropsOneThatCameBackPartial(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		keptHolder, lostHolder := peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)
		urls := urlsOf(3)
		h := openLoop(t, startClock(), loopOptions{redundancy: 3})
		h.reach(keptHolder, lostHolder)
		h.transfers.answers[keptHolder.Hash] = acceptedAnswer
		h.transfers.answers[lostHolder.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urls...)
		h.start(t)

		h.transfers.answers[lostHolder.Hash] = acceptedInPart(1)
		h.advance(refreshInterval)

		if got := urlsIn(h.transfers.offeredTo(keptHolder)); len(got) != 2*len(urls) {
			t.Fatalf("kept holder got %v, want the whole word partition twice", got)
		}
		want := []yacymodel.Hash{keptHolder.Hash}
		if got := h.holdersOf(t, wordPartition); !slices.Equal(got, want) {
			t.Fatalf("holders = %v, want only %v", got, want)
		}
	})
}

func TestLoopOffersARefreshThatLostEveryHolderAgainAfterTheRetryDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		holder := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(holder)
		h.transfers.answers[holder.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urlsOf(2)...)
		h.start(t)
		h.transfers.answers[holder.Hash] = acceptedInPart(1)
		h.advance(refreshInterval)

		h.transfers.answers[holder.Hash] = acceptedAnswer
		h.advance(retryDelay - time.Second)
		if got := len(h.transfers.offeredTo(holder)); got != 2 {
			t.Fatalf(
				"transfers before the retry delay = %d, want the delivery and one refresh",
				got,
			)
		}
		h.advance(time.Second)

		if got := len(h.transfers.offeredTo(holder)); got != 3 {
			t.Fatalf("transfers after the retry delay = %d, want the refresh sent again", got)
		}
		if got := h.holdersOf(t, wordPartition); !slices.Equal(got, []yacymodel.Hash{holder.Hash}) {
			t.Fatalf("holders = %v, want %v back", got, holder.Hash)
		}
	})
}

func TestLoopMovesTheNextRefreshOnceTheHoldersGotTheWordPartitionAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		holder := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(holder)
		h.transfers.answers[holder.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urlsOf(2)...)
		h.start(t)

		h.advance(refreshInterval)
		h.advance(time.Minute)

		if got := len(h.transfers.offeredTo(holder)); got != 2 {
			t.Fatalf("transfers to the holder = %d, want the delivery and one refresh", got)
		}
	})
}

func TestLoopDropsAStaleHolder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		staleHolder, responsiblePeer := peerAfter(wordPartition, 1<<40), peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(responsiblePeer)
		h.transfers.answers[responsiblePeer.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urlsOf(1)...)
		h.recordHolders(t, wordPartition, staleHolder.Hash)

		h.start(t)

		want := []yacymodel.Hash{responsiblePeer.Hash}
		if got := h.holdersOf(t, wordPartition); !slices.Equal(got, want) {
			t.Fatalf("holders = %v, want %v in place of the stale holder", got, want)
		}
		if h.observer.staleHoldersDropped != 1 {
			t.Fatalf("staleHoldersDropped = %d, want 1", h.observer.staleHoldersDropped)
		}
	})
}

func TestLoopHandsOffADeliveredWordPartition(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		closerPeer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{handoffEnabled: true})
		h.reach(closerPeer)
		h.transfers.answers[closerPeer.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urlsOf(3)...)

		h.start(t)

		if held := h.postingsOf(t, wordPartition); len(held) != 0 {
			t.Fatalf("postings held = %v, want the word partition handed off", held)
		}
		if h.observer.postingsHandedOff != 3 {
			t.Fatalf("postingsHandedOff = %d, want 3", h.observer.postingsHandedOff)
		}
		if holders := h.holdersOf(t, wordPartition); len(holders) != 0 {
			t.Fatalf("holders = %v, want them forgotten with the word partition", holders)
		}
	})
}

func TestLoopKeepsAWordPartitionThisNodeIsResponsibleFor(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{
			self:           thisNodeCloserThanEveryPeerOf(wordPartition),
			redundancy:     2,
			handoffEnabled: true,
		})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urlsOf(2)...)

		h.start(t)

		if held := h.postingsOf(t, wordPartition); len(held) != 2 {
			t.Fatalf("postings held = %v, want both kept: this node is responsible", held)
		}
		if h.observer.postingsHandedOff != 0 {
			t.Fatalf("postingsHandedOff = %d, want 0", h.observer.postingsHandedOff)
		}
	})
}

func TestLoopMergesTheSendsToOnePeerAndSplitsThemIntoTransfers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartitions := wordPartitionsIn(sectorOf(fixtureWordPartition()), 2)
		peer := peerAfter(wordPartitions[1], 1)
		h := openLoop(t, startClock(), loopOptions{
			self:             thisNodeFartherThan(wordPartitions[0]),
			redundancy:       2,
			postingsPerBatch: 2,
		})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.admit(t, wordPartitions[0], urlsOf(1)...)
		h.admit(t, wordPartitions[1], urlsOf(3)...)

		h.start(t)

		offered := h.transfers.offeredTo(peer)
		if len(offered) != 2 || len(offered[0].Postings) != 2 || len(offered[1].Postings) != 2 {
			t.Fatalf("offered = %+v, want two transfers of two postings", offered)
		}
		if offered[0].Postings[0].WordHash != wordPartitions[0].Word ||
			offered[0].Postings[1].WordHash != wordPartitions[1].Word {
			t.Fatalf(
				"first transfer = %+v, want postings of both word partitions",
				offered[0].Postings,
			)
		}
	})
}

func TestLoopSendsNoMoreTransfersToAPeerThatDeclinedOne(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{postingsPerBatch: 2})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = declinedAnswer
		h.admit(t, wordPartition, urlsOf(3)...)

		h.start(t)

		if offered := h.transfers.offeredTo(peer); len(offered) != 1 {
			t.Fatalf("offered = %+v, want one transfer before the peer declined", offered)
		}
	})
}

func TestLoopResumesTheWalkAfterTheLastBatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartitions := wordPartitionsIn(sectorOf(fixtureWordPartition()), 3)
		peer := peerAfter(wordPartitions[2], 1)
		h := openLoop(t, startClock(), loopOptions{
			self:             thisNodeFartherThan(wordPartitions[0]),
			redundancy:       2,
			postingsPerBatch: 2,
		})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedInPart(1)
		h.admit(t, wordPartitions[0], urlsOf(2)...)
		h.admit(t, wordPartitions[1], urlsOf(1)...)
		h.admit(t, wordPartitions[2], urlsOf(1)...)

		h.start(t)

		offersPerWord := offersPerWordIn(h.transfers.offered)
		for index, want := range []int{2, 1, 1} {
			if got := offersPerWord[wordPartitions[index].Word]; got != want {
				t.Fatalf("postings of word partition %d sent %d times, want %d: %v",
					index, got, want, offersPerWord)
			}
		}
	})
}

func TestLoopCountsUnreadDuePostings(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(peerAfter(wordPartition, 1))
		h.index.unread = errors.New("vault closed")
		h.admit(t, wordPartition, urlsOf(1)...)

		h.start(t)

		if h.observer.batchesAborted[string(distributionloop.AbortDuePostingsUnread)] != 1 {
			t.Fatalf("batchesAborted = %v, want one abort for unread due postings",
				h.observer.batchesAborted)
		}
		if len(h.transfers.offered) != 0 {
			t.Fatalf(
				"offered = %v, want no offers when due postings are unread",
				h.transfers.offered,
			)
		}
	})
}

func TestLoopCountsAnUnwrittenBatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urlsOf(1)...)
		h.transfers.onOffer = func(yacymodel.Seed) {
			if err := h.v.Close(); err != nil {
				t.Errorf("Close: %v", err)
			}
		}

		h.start(t)

		if h.observer.batchesAborted[string(distributionloop.AbortBatchUnwritten)] != 1 {
			t.Fatalf("batchesAborted = %v, want one abort for an unwritten batch",
				h.observer.batchesAborted)
		}
	})
}

func TestLoopReportsTheRingFractionOfEveryAcceptedPosting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.admit(t, wordPartition, urlsOf(2)...)

		h.start(t)

		want := positionOf(wordPartition).DistanceTo(
			yacymodel.DHTRingPositionOf(peer.Hash),
		).FractionOfDHTRing()
		if !slices.Equal(h.observer.holderRingFractions, []float64{want, want}) {
			t.Fatalf("holderRingFractions = %v, want %v for each accepted posting",
				h.observer.holderRingFractions, want)
		}
	})
}

func TestLoopReportsNoRingFractionsWithoutAcceptance(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		peer := peerAfter(wordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{})
		h.reach(peer)
		h.transfers.answers[peer.Hash] = declinedAnswer
		h.admit(t, wordPartition, urlsOf(1)...)

		h.start(t)

		if len(h.observer.holderRingFractions) != 0 {
			t.Fatalf("holderRingFractions = %v, want none when no peer accepted",
				h.observer.holderRingFractions)
		}
	})
}

func TestLoopDrainsABacklogLargerThanOneBatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const amountOfWordPartitions = 25
		h := openLoop(t, startClock(), loopOptions{redundancy: 2})
		peer := peerAfter(fixtureWordPartition(), 1)
		h.reach(peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		for word := range amountOfWordPartitions {
			h.admit(t, wordPartitionOf(word), urlsOf(1)...)
		}

		h.start(t)

		if got := h.amountOfUndeliveredPostings(t); got != 0 {
			t.Fatalf("undelivered postings = %d of %d, want none left",
				got, amountOfWordPartitions)
		}
		if h.observer.batchesCompleted < amountOfWordPartitions/postingsPerBatch {
			t.Fatalf("batchesCompleted = %d, want at least one per full batch",
				h.observer.batchesCompleted)
		}
	})
}

func TestLoopReportsThePeersAcceptingRemoteIndexPerRingSector(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartition := fixtureWordPartition()
		acceptingPeer, refusingPeer := peerAfter(wordPartition, 1), peerAfter(wordPartition, 1<<40)
		refusingPeer.Capabilities = yacymodel.Some(yacymodel.PeerCapabilities{})
		h := openLoop(t, startClock(), loopOptions{minReachablePeers: 3})
		h.reach(acceptingPeer, refusingPeer)

		h.start(t)

		want := yacymodel.SeedsPerDHTRingSector([]yacymodel.Seed{acceptingPeer})
		if !slices.Equal(h.observer.peersPerSector, want) {
			t.Fatalf("peers per sector = %v, want only %v counted", h.observer.peersPerSector, want)
		}
	})
}
