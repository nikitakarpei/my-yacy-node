package distributioncycle_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributioncycle"
)

func TestCycleRunOffersOnceThenStops(t *testing.T) {
	wordPartition := fixtureWordPartition()
	peer := peerAfter(wordPartition, 1)
	h := openCycle(t, startClock(), cycleOptions{})
	h.reach(peer)
	h.transfers.answers[peer.Hash] = acceptedAnswer
	h.admit(t, wordPartition, urlsOf(1)...)

	ctx, cancel := context.WithCancel(context.Background())
	h.transfers.onOffer = cancel
	h.cycle.Run(ctx)

	if len(h.transfers.offered) != 1 {
		t.Fatalf("offered = %v, want a single offer from the initial run", h.transfers.offered)
	}
}

func TestCycleSkipsWhenTooFewPeersReachable(t *testing.T) {
	wordPartition := fixtureWordPartition()
	h := openCycle(t, startClock(), cycleOptions{minReachablePeers: 2})
	h.reach(peerAfter(wordPartition, 1))
	h.admit(t, wordPartition, urlsOf(1)...)

	h.runCycle(t)

	if len(h.transfers.offered) != 0 {
		t.Fatalf("offered = %v, want no offers below the reachable-peer floor", h.transfers.offered)
	}
	if h.observer.cyclesSkipped[string(distributioncycle.SkipTooFewReachablePeers)] != 1 {
		t.Fatalf("cyclesSkipped = %v, want one skip below the reachable-peer floor",
			h.observer.cyclesSkipped)
	}
	if got := h.amountOfUndeliveredPostings(t); got != 1 {
		t.Fatalf("undelivered postings = %d, want 1 left untouched by a skipped cycle", got)
	}
}

func TestCycleReportsTheBacklogOnASkippedCycle(t *testing.T) {
	clk := startClock()
	delivered, undelivered := wordPartitionOf(0), wordPartitionOf(1)
	h := openCycle(t, clk, cycleOptions{minReachablePeers: 2})
	h.admit(t, undelivered, urlsOf(3)...)
	h.recordHolders(t, delivered, peerAfter(delivered, 1).Hash)
	clk.at = clk.at.Add(refreshInterval + time.Minute)

	h.runCycle(t)

	if h.observer.undeliveredPostings != 3 || h.observer.longestRefreshLateness != time.Minute {
		t.Fatalf("undelivered postings = %d, longest refresh lateness = %v, want 3 and 1m",
			h.observer.undeliveredPostings, h.observer.longestRefreshLateness)
	}
}

func TestCycleReportsNoRefreshLatenessBeforeTheRefreshTime(t *testing.T) {
	clk := startClock()
	delivered := wordPartitionOf(0)
	h := openCycle(t, clk, cycleOptions{minReachablePeers: 2})
	h.recordHolders(t, delivered, peerAfter(delivered, 1).Hash)
	h.observer.longestRefreshLateness = time.Hour

	h.runCycle(t)

	if h.observer.longestRefreshLateness != 0 {
		t.Fatalf("longest refresh lateness = %v, want 0", h.observer.longestRefreshLateness)
	}
}

func TestCycleDeliversANewPostingToEveryResponsiblePeer(t *testing.T) {
	wordPartition := fixtureWordPartition()
	peers := []yacymodel.Seed{peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)}
	h := openCycle(t, startClock(), cycleOptions{redundancy: 3})
	h.reach(peers...)
	for _, peer := range peers {
		h.transfers.answers[peer.Hash] = acceptedAnswer
	}
	h.admit(t, wordPartition, urlsOf(1)...)

	h.runCycle(t)

	for _, peer := range peers {
		if got := urlsIn(h.transfers.offeredTo(peer)); !slices.Equal(got, urlsOf(1)) {
			t.Fatalf("%v got %v, want the new posting", peer.Hash, got)
		}
	}
	if got := h.amountOfUndeliveredPostings(t); got != 0 {
		t.Fatalf("undelivered postings = %d, want the new posting delivered", got)
	}
	if got, want := h.holdersOf(
		t,
		wordPartition,
	), []yacymodel.Hash{
		peers[0].Hash,
		peers[1].Hash,
	}; !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("holders = %v, want %v", got, want)
	}
}

func TestCycleSendsTheWholeWordPartitionToAMissingHolder(t *testing.T) {
	wordPartition := fixtureWordPartition()
	holder, missingHolder := peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)
	urls := urlsOf(4)
	h := openCycle(t, startClock(), cycleOptions{redundancy: 3})
	h.transfers.answers[holder.Hash] = acceptedAnswer
	h.transfers.answers[missingHolder.Hash] = acceptedAnswer
	h.reach(holder)
	h.admit(t, wordPartition, urls[:3]...)
	h.runCycle(t)

	h.reach(holder, missingHolder)
	h.admit(t, wordPartition, urls[3])
	h.runCycle(t)

	if got := urlsIn(h.transfers.offeredTo(holder)); !slices.Equal(got, sortedURLs(urls)) {
		t.Fatalf("holder got %v, want its first postings and then only the new one", got)
	}
	if got := urlsIn(h.transfers.offeredTo(missingHolder)); !slices.Equal(got, sortedURLs(urls)) {
		t.Fatalf("missing holder got %v, want the whole word partition %v", got, sortedURLs(urls))
	}
	if got := h.holdersOf(t, wordPartition); !slices.Contains(got, missingHolder.Hash) {
		t.Fatalf("holders = %v, want %v among them", got, missingHolder.Hash)
	}
	if got := h.amountOfUndeliveredPostings(t); got != 0 {
		t.Fatalf("undelivered postings = %d, want none", got)
	}
}

func TestCycleLetsAWordPartitionWaitForAResponsiblePeerInCooldown(t *testing.T) {
	clk := startClock()
	wordPartition := fixtureWordPartition()
	holder, peerInCooldown := peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)
	h := openCycle(t, clk, cycleOptions{redundancy: 3, cooldown: time.Hour})
	h.reach(holder, peerInCooldown)
	h.transfers.answers[holder.Hash] = acceptedAnswer
	h.transfers.answers[peerInCooldown.Hash] = declinedAnswer
	h.admit(t, wordPartition, urlsOf(2)...)
	h.runCycle(t)

	clk.at = clk.at.Add(time.Minute)
	h.runCycle(t)

	if len(h.transfers.offered) != 2 {
		t.Fatalf(
			"offered = %v, want only the first two: one peer is in cooldown",
			h.transfers.offered,
		)
	}
	if got := h.amountOfUndeliveredPostings(t); got != 2 {
		t.Fatalf("undelivered postings = %d, want 2 waiting for the peer in cooldown", got)
	}

	clk.at = clk.at.Add(time.Hour)
	h.transfers.answers[peerInCooldown.Hash] = acceptedAnswer
	h.runCycle(t)

	if got := urlsIn(h.transfers.offeredTo(peerInCooldown)); len(got) != 4 {
		t.Fatalf("peer after its cooldown got %v, want the word partition again", got)
	}
	if got := h.amountOfUndeliveredPostings(t); got != 0 {
		t.Fatalf("undelivered postings = %d, want none once the cooldown ended", got)
	}
}

func TestCycleRecordsNoHolderOnADeclinedOffer(t *testing.T) {
	wordPartition := fixtureWordPartition()
	peer := peerAfter(wordPartition, 1)
	h := openCycle(t, startClock(), cycleOptions{})
	h.reach(peer)
	h.transfers.answers[peer.Hash] = declinedAnswer
	h.admit(t, wordPartition, urlsOf(1)...)

	h.runCycle(t)

	if holders := h.holdersOf(t, wordPartition); len(holders) != 0 {
		t.Fatalf("holders = %v, want none after a declined offer", holders)
	}
	if got := h.amountOfUndeliveredPostings(t); got != 1 {
		t.Fatalf("undelivered postings = %d, want 1", got)
	}
}

func TestCycleRefreshesTheHoldersAndDropsOneThatCameBackPartial(t *testing.T) {
	clk := startClock()
	wordPartition := fixtureWordPartition()
	keptHolder, lostHolder := peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)
	urls := urlsOf(3)
	h := openCycle(t, clk, cycleOptions{redundancy: 3})
	h.reach(keptHolder, lostHolder)
	h.transfers.answers[keptHolder.Hash] = acceptedAnswer
	h.transfers.answers[lostHolder.Hash] = acceptedAnswer
	h.admit(t, wordPartition, urls...)
	h.runCycle(t)

	clk.at = clk.at.Add(refreshInterval)
	h.transfers.answers[lostHolder.Hash] = acceptedInPart(1)
	h.runCycle(t)

	if got := urlsIn(h.transfers.offeredTo(keptHolder)); len(got) != 2*len(urls) {
		t.Fatalf("kept holder got %v, want the whole word partition twice", got)
	}
	if got, want := h.holdersOf(
		t,
		wordPartition,
	), []yacymodel.Hash{
		keptHolder.Hash,
	}; !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("holders = %v, want only %v", got, want)
	}
}

func TestCycleOffersARefreshThatLostEveryHolderAgainAfterTheRetryDelay(t *testing.T) {
	clk := startClock()
	wordPartition := fixtureWordPartition()
	holder := peerAfter(wordPartition, 1)
	h := openCycle(t, clk, cycleOptions{})
	h.reach(holder)
	h.transfers.answers[holder.Hash] = acceptedAnswer
	h.admit(t, wordPartition, urlsOf(2)...)
	h.runCycle(t)
	clk.at = clk.at.Add(refreshInterval)
	h.transfers.answers[holder.Hash] = acceptedInPart(1)
	h.runCycle(t)

	h.transfers.answers[holder.Hash] = acceptedAnswer
	clk.at = clk.at.Add(retryDelay - time.Second)
	h.runCycle(t)
	if got := len(h.transfers.offeredTo(holder)); got != 2 {
		t.Fatalf("transfers before the retry delay = %d, want the delivery and one refresh", got)
	}
	clk.at = clk.at.Add(time.Second)
	h.runCycle(t)

	if got := len(h.transfers.offeredTo(holder)); got != 3 {
		t.Fatalf("transfers after the retry delay = %d, want the refresh sent again", got)
	}
	if got := h.holdersOf(t, wordPartition); !slices.Equal(got, []yacymodel.Hash{holder.Hash}) {
		t.Fatalf("holders = %v, want %v back", got, holder.Hash)
	}
}

func TestCycleMovesTheNextRefreshOnceTheHoldersGotTheWordPartitionAgain(t *testing.T) {
	clk := startClock()
	wordPartition := fixtureWordPartition()
	holder := peerAfter(wordPartition, 1)
	h := openCycle(t, clk, cycleOptions{})
	h.reach(holder)
	h.transfers.answers[holder.Hash] = acceptedAnswer
	h.admit(t, wordPartition, urlsOf(2)...)
	h.runCycle(t)

	clk.at = clk.at.Add(refreshInterval)
	h.runCycle(t)
	clk.at = clk.at.Add(time.Minute)
	h.runCycle(t)

	if got := len(h.transfers.offeredTo(holder)); got != 2 {
		t.Fatalf("transfers to the holder = %d, want the delivery and one refresh", got)
	}
}

func TestCycleDropsAStaleHolder(t *testing.T) {
	wordPartition := fixtureWordPartition()
	staleHolder, responsiblePeer := peerAfter(wordPartition, 1<<40), peerAfter(wordPartition, 1)
	h := openCycle(t, startClock(), cycleOptions{})
	h.reach(responsiblePeer)
	h.transfers.answers[responsiblePeer.Hash] = acceptedAnswer
	h.admit(t, wordPartition, urlsOf(1)...)
	h.recordHolders(t, wordPartition, staleHolder.Hash)

	h.runCycle(t)

	if got, want := h.holdersOf(
		t,
		wordPartition,
	), []yacymodel.Hash{
		responsiblePeer.Hash,
	}; !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("holders = %v, want %v in place of the stale holder", got, want)
	}
	if h.observer.staleReplicasDropped != 1 {
		t.Fatalf("staleReplicasDropped = %d, want 1", h.observer.staleReplicasDropped)
	}
}

func TestCycleHandsOffADeliveredWordPartition(t *testing.T) {
	wordPartition := fixtureWordPartition()
	closerPeer := peerAfter(wordPartition, 1)
	h := openCycle(t, startClock(), cycleOptions{handoffEnabled: true})
	h.reach(closerPeer)
	h.transfers.answers[closerPeer.Hash] = acceptedAnswer
	h.admit(t, wordPartition, urlsOf(3)...)

	h.runCycle(t)

	if held := h.postingsOf(t, wordPartition); len(held) != 0 {
		t.Fatalf("postings held = %v, want the word partition handed off", held)
	}
	if h.observer.postingsHandedOff != 3 {
		t.Fatalf("postingsHandedOff = %d, want 3", h.observer.postingsHandedOff)
	}
	if holders := h.holdersOf(t, wordPartition); len(holders) != 0 {
		t.Fatalf("holders = %v, want them forgotten with the word partition", holders)
	}
}

func TestCycleKeepsAWordPartitionThisNodeIsResponsibleFor(t *testing.T) {
	wordPartition := fixtureWordPartition()
	peer := peerAfter(wordPartition, 1)
	h := openCycle(t, startClock(), cycleOptions{
		self:           thisNodeCloserThanEveryPeerOf(wordPartition),
		redundancy:     2,
		handoffEnabled: true,
	})
	h.reach(peer)
	h.transfers.answers[peer.Hash] = acceptedAnswer
	h.admit(t, wordPartition, urlsOf(2)...)

	h.runCycle(t)

	if held := h.postingsOf(t, wordPartition); len(held) != 2 {
		t.Fatalf("postings held = %v, want both kept: this node is responsible", held)
	}
	if h.observer.postingsHandedOff != 0 {
		t.Fatalf("postingsHandedOff = %d, want 0", h.observer.postingsHandedOff)
	}
}

func TestCycleMergesTheSendsToOnePeerAndSplitsThemIntoTransfers(t *testing.T) {
	wordPartitions := wordPartitionsIn(sectorOf(fixtureWordPartition()), 2)
	peer := peerAfter(wordPartitions[1], 1)
	h := openCycle(t, startClock(), cycleOptions{
		self:             thisNodeFartherThan(wordPartitions[0]),
		redundancy:       2,
		postingsPerBatch: 2,
	})
	h.reach(peer)
	h.transfers.answers[peer.Hash] = acceptedAnswer
	h.admit(t, wordPartitions[0], urlsOf(1)...)
	h.admit(t, wordPartitions[1], urlsOf(3)...)

	h.runCycle(t)

	offered := h.transfers.offeredTo(peer)
	if len(offered) != 2 || len(offered[0].Postings) != 2 || len(offered[1].Postings) != 2 {
		t.Fatalf("offered = %+v, want two transfers of two postings", offered)
	}
	if offered[0].Postings[0].WordHash != wordPartitions[0].Word ||
		offered[0].Postings[1].WordHash != wordPartitions[1].Word {
		t.Fatalf("first transfer = %+v, want postings of both word partitions", offered[0].Postings)
	}
}

func TestCycleResumesTheWalkAfterTheLastBatch(t *testing.T) {
	wordPartitions := wordPartitionsIn(sectorOf(fixtureWordPartition()), 3)
	peer := peerAfter(wordPartitions[2], 1)
	h := openCycle(t, startClock(), cycleOptions{
		self:             thisNodeFartherThan(wordPartitions[0]),
		redundancy:       2,
		postingsPerBatch: 2,
	})
	h.reach(peer)
	h.transfers.answers[peer.Hash] = acceptedInPart(1)
	h.admit(t, wordPartitions[0], urlsOf(2)...)
	h.admit(t, wordPartitions[1], urlsOf(1)...)
	h.admit(t, wordPartitions[2], urlsOf(1)...)

	h.runCycle(t)

	offersPerWord := map[yacymodel.Hash]int{}
	for _, call := range h.transfers.offered {
		for _, posting := range call.Postings {
			offersPerWord[posting.WordHash]++
		}
	}
	for index, want := range []int{2, 1, 1} {
		if got := offersPerWord[wordPartitions[index].Word]; got != want {
			t.Fatalf("postings of word partition %d sent %d times, want %d: %v",
				index, got, want, offersPerWord)
		}
	}
}

func TestCycleCountsUnreadDuePostings(t *testing.T) {
	wordPartition := fixtureWordPartition()
	h := openCycle(t, startClock(), cycleOptions{})
	h.reach(peerAfter(wordPartition, 1))
	h.index.unread = errors.New("vault closed")
	h.admit(t, wordPartition, urlsOf(1)...)

	h.runCycle(t)

	if h.observer.batchesAborted[string(distributioncycle.AbortDuePostingsUnread)] != 1 {
		t.Fatalf("batchesAborted = %v, want one abort for unread due postings",
			h.observer.batchesAborted)
	}
	if len(h.transfers.offered) != 0 {
		t.Fatalf("offered = %v, want no offers when due postings are unread", h.transfers.offered)
	}
}

func TestCycleCountsAnUnwrittenBatch(t *testing.T) {
	wordPartition := fixtureWordPartition()
	peer := peerAfter(wordPartition, 1)
	h := openCycle(t, startClock(), cycleOptions{})
	h.reach(peer)
	h.transfers.answers[peer.Hash] = acceptedAnswer
	h.admit(t, wordPartition, urlsOf(1)...)
	h.transfers.onOffer = func() {
		if err := h.v.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}

	h.runCycle(t)

	if h.observer.batchesAborted[string(distributioncycle.AbortBatchUnwritten)] != 1 {
		t.Fatalf("batchesAborted = %v, want one abort for an unwritten batch",
			h.observer.batchesAborted)
	}
}

func TestCycleReportsTheRingFractionOfEveryAcceptedPosting(t *testing.T) {
	wordPartition := fixtureWordPartition()
	peer := peerAfter(wordPartition, 1)
	h := openCycle(t, startClock(), cycleOptions{})
	h.reach(peer)
	h.transfers.answers[peer.Hash] = acceptedAnswer
	h.admit(t, wordPartition, urlsOf(2)...)

	h.runCycle(t)

	want := positionOf(wordPartition).DistanceTo(
		yacymodel.DHTRingPositionOf(peer.Hash),
	).FractionOfDHTRing()
	if !slices.Equal(h.observer.replicaRingFractions, []float64{want, want}) {
		t.Fatalf("replicaRingFractions = %v, want %v for each accepted posting",
			h.observer.replicaRingFractions, want)
	}
}

func TestCycleReportsNoRingFractionsWithoutAcceptance(t *testing.T) {
	wordPartition := fixtureWordPartition()
	peer := peerAfter(wordPartition, 1)
	h := openCycle(t, startClock(), cycleOptions{})
	h.reach(peer)
	h.transfers.answers[peer.Hash] = declinedAnswer
	h.admit(t, wordPartition, urlsOf(1)...)

	h.runCycle(t)

	if len(h.observer.replicaRingFractions) != 0 {
		t.Fatalf("replicaRingFractions = %v, want none when no peer accepted",
			h.observer.replicaRingFractions)
	}
}

func TestCycleDrainsABacklogLargerThanOneBatch(t *testing.T) {
	h, backlog := openBacklog(t, startClock(), cycleOptions{})

	h.runCycle(t)

	if got := h.amountOfUndeliveredPostings(t); got != 0 {
		t.Fatalf("undelivered postings = %d of %d, want none left after one cycle", got, backlog)
	}
}

func TestCycleStopsDrainingWhenTheBudgetIsSpent(t *testing.T) {
	clk := startClock()
	h, _ := openBacklog(t, clk, cycleOptions{drainBudget: 30 * time.Second})
	h.transfers.onOffer = func() { clk.at = clk.at.Add(30 * time.Second) }

	h.runCycle(t)

	if len(h.transfers.offered) != 1 {
		t.Fatalf(
			"offered %d transfers, want the drain to stop after the batch that spends the budget",
			len(h.transfers.offered),
		)
	}
	if h.amountOfUndeliveredPostings(t) == 0 {
		t.Fatal("the spent budget left no posting undelivered, want the backlog unfinished")
	}
}

func openBacklog(t *testing.T, clk *clock, opts cycleOptions) (*cycleHarness, int) {
	t.Helper()

	const amountOfWordPartitions = 25
	opts.redundancy = 2
	h := openCycle(t, clk, opts)
	peer := peerAfter(fixtureWordPartition(), 1)
	h.reach(peer)
	h.transfers.answers[peer.Hash] = acceptedAnswer
	for word := range amountOfWordPartitions {
		h.admit(t, wordPartitionOf(word), urlsOf(1)...)
	}

	return h, amountOfWordPartitions
}
