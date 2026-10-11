package distributioncycle_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributioncycle"
)

func TestCycleRunOffersOnceThenStops(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings: postings,
		roster:   fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
	})
	h.transfers.answers[peer] = acceptedAnswer

	store(t, h.v, h.schedule, word, url)

	ctx, cancel := context.WithCancel(context.Background())
	h.transfers.onOffer = cancel
	h.cycle.Run(ctx)

	if len(h.transfers.offered) != 1 {
		t.Fatalf("offered = %v, want a single offer from the initial run", h.transfers.offered)
	}
}

func TestCycleSkipsWhenTooFewPeersReachable(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings:          postings,
		roster:            fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
		minReachablePeers: 2,
	})

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)

	if len(h.transfers.offered) != 0 {
		t.Fatalf(
			"offered = %v, want no offers while below the reachable-peer floor",
			h.transfers.offered,
		)
	}
	if h.observer.cyclesSkipped[string(distributioncycle.SkipTooFewReachablePeers)] != 1 {
		t.Fatalf("cyclesSkipped = %v, want one skip below the reachable-peer floor",
			h.observer.cyclesSkipped)
	}

	due := h.duePostings(t, 10)
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] left untouched by a skipped cycle", due)
	}
}

func TestCycleReportsLongestOfferLatenessOnSkippedCycle(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	clk := &clock{at: now}
	h := openCycle(t, clk, cycleOptions{
		postings:          postings,
		roster:            fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
		minReachablePeers: 2,
	})

	store(t, h.v, h.schedule, word, url)

	clk.at = now.Add(90 * time.Second)
	h.runCycle(t)

	if len(h.transfers.offered) != 0 {
		t.Fatalf("offered = %v, want no offers on a skipped cycle", h.transfers.offered)
	}
	if got := h.observer.longestOfferLateness[shortfall]; got != 90*time.Second {
		t.Fatalf("shortfall longestOfferLateness = %v, want 90s", got)
	}
}

func TestCycleDropsStaleReplicaFromLedger(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	stalePeer, fresh := yacymodel.WordHash("stale"), yacymodel.WordHash("fresh")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings: postings,
		roster:   fakeRoster{reachable: []yacymodel.Seed{seed(fresh)}},
	})
	h.transfers.answers[fresh] = acceptedAnswer

	store(t, h.v, h.schedule, word, url)
	recordAccepted(t, h, stalePeer, fakePosting(word, url))

	h.runCycle(t)

	replicas := holdersOf(t, h, word, url)
	for _, replica := range replicas {
		if replica == stalePeer {
			t.Fatalf("replicas = %v, want %v dropped", replicas, stalePeer)
		}
	}
	if h.observer.staleReplicasDropped != 1 {
		t.Fatalf("staleReplicasDropped = %v, want 1", h.observer.staleReplicasDropped)
	}
}

func TestCycleKeepsRecentlyReachableReplicaFromLedger(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	recentPeer := yacymodel.WordHash("recent")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	credible := fakeRoster{recentlyReachable: map[yacymodel.Hash]struct{}{recentPeer: {}}}
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings: postings,
		roster:   credible,
	})

	store(t, h.v, h.schedule, word, url)
	recordAccepted(t, h, recentPeer, fakePosting(word, url))

	h.runCycle(t)

	replicas := holdersOf(t, h, word, url)
	if len(replicas) != 1 || replicas[0] != recentPeer {
		t.Fatalf(
			"replicas = %v, want %v kept while its confirmation is still credible",
			replicas, recentPeer,
		)
	}
	if h.observer.staleReplicasDropped != 0 {
		t.Fatalf("staleReplicasDropped = %v, want 0", h.observer.staleReplicasDropped)
	}
	if len(h.transfers.offered) != 0 {
		t.Fatalf(
			"offered = %v, want no offer once the credible peer already holds a replica",
			h.transfers.offered,
		)
	}
}

func TestCycleReschedulesAcceptedPostingAtTheLongestOfferInterval(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings:   postings,
		roster:     fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
		redundancy: 2,
		self:       thisNodeCloserThanEveryPeer(),
	})
	h.transfers.answers[peer] = acceptedAnswer

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)

	due := h.duePostings(t, 10)
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due right after redundancy is met", due)
	}
}

func TestCycleRetriesRejectedPostingAtBackoffInterval(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	clk := &clock{at: now}
	h := openCycle(t, clk, cycleOptions{
		postings: postings,
		roster:   fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
	})
	h.transfers.answers[peer] = declinedAnswer

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)

	due := h.duePostings(t, 10)
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due immediately after a rejected offer", due)
	}

	clk.at = now.Add(h.offerInterval.Shortest + time.Second)
	due = h.duePostings(t, 10)
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] once the shortest offer interval has elapsed", due)
	}
}

func TestCycleHonoursThePauseAPeerRequests(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	clk := &clock{at: now}
	h := openCycle(t, clk, cycleOptions{
		postings: postings,
		roster:   fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
	})
	h.transfers.answers[peer] = fakeAnswer{requestedPause: 5 * time.Minute}

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)

	clk.at = now.Add(h.offerInterval.Shortest + time.Second)
	due := h.duePostings(t, 10)
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due before the peer's pause elapses", due)
	}

	clk.at = now.Add(5*time.Minute + time.Second)
	due = h.duePostings(t, 10)
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] once the peer's pause has elapsed", due)
	}
}

func TestCycleReschedulesUnofferedPostingAtBackoffInterval(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	clk := &clock{at: now}
	h := openCycle(t, clk, cycleOptions{postings: postings, redundancy: 2})

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)

	if len(h.transfers.offered) != 0 {
		t.Fatalf("offered = %v, want no offers for an unoffered posting", h.transfers.offered)
	}

	due := h.duePostings(t, 10)
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due immediately after stalling", due)
	}

	clk.at = now.Add(h.offerInterval.Shortest + time.Second)
	due = h.duePostings(t, 10)
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] once the shortest offer interval has elapsed", due)
	}
}

func TestCycleReschedulesAlreadySatisfiedPostingAtTheLongestOfferInterval(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	clk := &clock{at: now}
	h := openCycle(t, clk, cycleOptions{
		postings:   postings,
		roster:     fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
		redundancy: 2,
		self:       thisNodeCloserThanEveryPeer(),
	})

	store(t, h.v, h.schedule, word, url)
	recordAccepted(t, h, peer, fakePosting(word, url))

	h.runCycle(t)

	if len(h.transfers.offered) != 1 || h.transfers.offered[0].Peer.Hash != peer {
		t.Fatalf(
			"offered = %v, want the posting renewed with the peer that holds it",
			h.transfers.offered,
		)
	}

	clk.at = now.Add(h.offerInterval.Longest - time.Second)
	due := h.duePostings(t, 10)
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due before the longest offer interval elapses", due)
	}
}

func TestCycleRecordsReplicaOnAcceptedOffer(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings:   postings,
		roster:     fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
		redundancy: 2,
		self:       thisNodeCloserThanEveryPeer(),
	})
	h.transfers.answers[peer] = acceptedAnswer

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)

	replicas := holdersOf(t, h, word, url)
	if len(replicas) != 1 || replicas[0] != peer {
		t.Fatalf("replicas = %v, want [%v]", replicas, peer)
	}
}

func TestCycleRecordsNoReplicaOfAPostingPurgedWhileOffered(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings:   postings,
		roster:     fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
		redundancy: 2,
		self:       thisNodeCloserThanEveryPeer(),
	})
	h.transfers.answers[peer] = acceptedAnswer
	h.transfers.onOffer = func() {
		if err := h.v.Update(context.Background(), func(tx *vault.Txn) error {
			return h.schedule.PostingPurged(tx, fakePosting(word, url))
		}); err != nil {
			t.Errorf("PostingPurged: %v", err)
		}
	}

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)

	if replicas := holdersOf(t, h, word, url); len(replicas) != 0 {
		t.Fatalf("replicas = %v, want none for a posting purged while offered", replicas)
	}
}

func TestCycleReschedulesAtLongestBackoffAcrossPeers(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peerA, peerB := yacymodel.WordHash("a"), yacymodel.WordHash("b")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	clk := &clock{at: now}
	h := openCycle(t, clk, cycleOptions{
		postings:   postings,
		roster:     fakeRoster{reachable: []yacymodel.Seed{seed(peerA), seed(peerB)}},
		redundancy: 2,
	})
	h.transfers.answers[peerA] = fakeAnswer{requestedPause: time.Minute}
	h.transfers.answers[peerB] = fakeAnswer{requestedPause: 5 * time.Minute}

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)

	clk.at = now.Add(time.Minute + time.Second)
	due := h.duePostings(t, 10)
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due before the longer pause elapses", due)
	}

	clk.at = now.Add(5*time.Minute + time.Second)
	due = h.duePostings(t, 10)
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] once the longer pause has elapsed", due)
	}
}

func TestCycleCountsUnreadPostingOffer(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	h := openCycle(t, &clock{at: now}, cycleOptions{
		roster:      fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
		postingsErr: errors.New("vault closed"),
	})

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)

	if h.observer.batchesAborted[string(distributioncycle.AbortDuePostingsUnread)] != 1 {
		t.Fatalf("batchesAborted = %v, want one abort for an unread offer",
			h.observer.batchesAborted)
	}
	if len(h.transfers.offered) != 0 {
		t.Fatalf("offered = %v, want no offers when offer is unread", h.transfers.offered)
	}
}

func TestCycleCountsAnUnwrittenBatch(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings: map[yacymodel.Hash]yacymodel.RWIPosting{
			fakePostingKey(word, url): fakePosting(word, url),
		},
		roster: fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
	})
	h.transfers.answers[peer] = acceptedAnswer
	h.transfers.onOffer = func() {
		if err := h.v.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)

	if h.observer.batchesAborted[string(distributioncycle.AbortBatchUnwritten)] != 1 {
		t.Fatalf("batchesAborted = %v, want one abort for an unwritten batch",
			h.observer.batchesAborted)
	}
}

func TestCycleReportsAnEmptyScheduleAsNoLateness(t *testing.T) {
	now := time.Unix(1000, 0)
	peer := yacymodel.WordHash("peer")
	h := openCycle(t, &clock{at: now}, cycleOptions{
		roster: fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
	})
	h.observer.longestOfferLateness[shortfall] = time.Hour

	h.runCycle(t)

	if got := h.observer.longestOfferLateness[shortfall]; got != 0 {
		t.Fatalf("shortfall longestOfferLateness = %v, want 0 while nothing is scheduled", got)
	}
	if got := h.observer.sumOfScheduledPostings(); got != 0 {
		t.Fatalf("scheduledPostings = %d, want 0 while nothing is scheduled", got)
	}
}

func TestCycleReportsScheduledPostings(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings: postings,
		roster:   fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
	})

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)

	if got := h.observer.sumOfScheduledPostings(); got != 1 {
		t.Fatalf("scheduledPostings = %d, want 1 for the single stored posting", got)
	}
}

func TestCycleRecordsNoReplicaOnRefusedOffer(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings: postings,
		roster:   fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
	})
	h.transfers.answers[peer] = declinedAnswer

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)

	replicas := holdersOf(t, h, word, url)
	if len(replicas) != 0 {
		t.Fatalf("replicas = %v, want none after a refused offer", replicas)
	}
}

func TestCycleOffersToAnotherPeerAfterAnUnreachableAnswer(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{
		seed(yacymodel.WordHash("p1")),
		seed(yacymodel.WordHash("p2")),
	}
	clk := &clock{at: now}
	h := openCycle(t, clk, cycleOptions{
		postings: postings,
		roster:   fakeRoster{reachable: peers},
		cooldown: time.Hour,
	})
	for _, peer := range peers {
		h.transfers.answers[peer.Hash] = declinedAnswer
	}

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)
	clk.at = clk.at.Add(2 * h.offerInterval.Shortest)
	h.runCycle(t)

	if len(h.transfers.offered) != 2 {
		t.Fatalf("offered = %v, want one offer from each cycle", h.transfers.offered)
	}
	if h.transfers.offered[0].Peer.Hash == h.transfers.offered[1].Peer.Hash {
		t.Fatalf(
			"both offers went to %v, want the second cycle to try another peer",
			h.transfers.offered[0].Peer.Hash,
		)
	}
}

func TestCycleDoublesTheWaitOfAPostingThatKeepsMissingRedundancy(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	clk := &clock{at: now}
	h := openCycle(t, clk, cycleOptions{postings: postings, redundancy: 2})

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)
	clk.at = now.Add(h.offerInterval.Shortest)
	h.runCycle(t)

	clk.at = now.Add(2*h.offerInterval.Shortest + time.Second)
	due := h.duePostings(t, 10)
	if len(due) != 0 {
		t.Fatalf("due = %v, want none: the second miss doubles the wait", due)
	}

	clk.at = now.Add(3*h.offerInterval.Shortest + time.Second)
	due = h.duePostings(t, 10)
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] once the doubled wait has elapsed", due)
	}
}

func TestCycleHandsOffPostingAcceptedByACloserPeer(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings: postings,
		roster:   fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
	})
	h.transfers.answers[peer] = acceptedAnswer

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)

	if _, found := h.postings.postings[fakePostingKey(word, url)]; found {
		t.Fatal("posting still stored, want it handed off to the closer peer")
	}
	if h.observer.postingsHandedOff != 1 {
		t.Fatalf("postingsHandedOff = %d, want 1", h.observer.postingsHandedOff)
	}

	h.runCycle(t)

	if got := h.observer.sumOfScheduledPostings(); got != 0 {
		t.Fatalf("scheduled = %d, want none: a handed-off posting leaves the schedule", got)
	}
	replicas := holdersOf(t, h, word, url)
	if len(replicas) != 0 {
		t.Fatalf("replicas = %v, want none: a handed-off posting leaves the ledger", replicas)
	}
}

func TestCycleKeepsPostingThisNodeIsResponsibleFor(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings:   postings,
		roster:     fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
		redundancy: 2,
		self:       thisNodeCloserThanEveryPeer(),
	})
	h.transfers.answers[peer] = acceptedAnswer

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)

	if _, found := h.postings.postings[fakePostingKey(word, url)]; !found {
		t.Fatal("posting handed off, want it kept: the DHT makes this node responsible for it")
	}
	if h.observer.postingsHandedOff != 0 {
		t.Fatalf("postingsHandedOff = %d, want 0", h.observer.postingsHandedOff)
	}
}

func TestCycleKeepsPostingAcceptedOnlyByAFartherPeer(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	closer, farther := yacymodel.WordHash("second"), yacymodel.WordHash("p1")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	clk := &clock{at: now}
	h := openCycle(t, clk, cycleOptions{
		postings: postings,
		roster:   fakeRoster{reachable: []yacymodel.Seed{seed(closer), seed(farther)}},
		self:     thisNodeFartherThanTheClosestPeer(),
		cooldown: time.Hour,
	})
	h.transfers.answers[closer] = declinedAnswer
	h.transfers.answers[farther] = acceptedAnswer

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)
	clk.at = now.Add(2 * h.offerInterval.Shortest)
	h.runCycle(t)

	if _, found := h.postings.postings[fakePostingKey(word, url)]; !found {
		t.Fatal("posting handed off, want it kept: only a farther peer holds it")
	}
	if h.observer.postingsHandedOff != 0 {
		t.Fatalf("postingsHandedOff = %d, want 0", h.observer.postingsHandedOff)
	}
}

func TestCycleReportsRingFractionOfAcceptedReplica(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings: postings,
		roster:   fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
	})
	h.transfers.answers[peer] = acceptedAnswer

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)

	if len(h.observer.replicaRingFractions) != 1 {
		t.Fatalf(
			"replicaRingFractions = %v, want one entry per accepted replica",
			h.observer.replicaRingFractions,
		)
	}

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}
	postingPosition := yacymodel.DHTRingPositionOfPosting(fakePosting(word, url), partitions)
	want := postingPosition.DistanceTo(yacymodel.DHTRingPositionOf(peer)).FractionOfDHTRing()
	if h.observer.replicaRingFractions[0] != want {
		t.Errorf(
			"replicaRingFraction = %v, want %v",
			h.observer.replicaRingFractions[0], want,
		)
	}
}

func TestCycleReportsNoRingFractionsWithoutAcceptance(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings: postings,
		roster:   fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
	})
	h.transfers.answers[peer] = declinedAnswer

	store(t, h.v, h.schedule, word, url)

	h.runCycle(t)

	if len(h.observer.replicaRingFractions) != 0 {
		t.Fatalf(
			"replicaRingFractions = %v, want none when no peer accepted",
			h.observer.replicaRingFractions,
		)
	}
}

func TestCycleDrainsABacklogLargerThanOneBatch(t *testing.T) {
	now := time.Unix(1000, 0)
	peer := yacymodel.WordHash("peer")
	backlog := 25
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings: backlogPostings(backlog),
		roster:   fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
	})
	h.transfers.answers[peer] = acceptedAnswer

	storeBacklog(t, h, backlog)

	h.runCycle(t)

	due := h.duePostings(t, backlog)
	if len(due) != 0 {
		t.Fatalf("due = %d postings, want none left after one cycle", len(due))
	}
}

func TestCycleStopsDrainingWhenTheBudgetIsSpent(t *testing.T) {
	now := time.Unix(1000, 0)
	peer := yacymodel.WordHash("peer")
	backlog := 25
	clk := &clock{at: now}
	h := openCycle(t, clk, cycleOptions{
		postings:    backlogPostings(backlog),
		roster:      fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
		drainBudget: 30 * time.Second,
	})
	h.transfers.answers[peer] = acceptedAnswer
	h.transfers.onOffer = func() { clk.at = clk.at.Add(30 * time.Second) }

	storeBacklog(t, h, backlog)

	h.runCycle(t)

	if len(h.transfers.offered) != 1 {
		t.Fatalf(
			"offered %d batches, want the drain to stop after the batch that spends the budget",
			len(h.transfers.offered),
		)
	}
	due := h.duePostings(t, backlog)
	if len(due) == 0 {
		t.Fatalf("the spent budget left no posting due, want the backlog unfinished")
	}
}

func backlogPostings(count int) map[yacymodel.Hash]yacymodel.RWIPosting {
	postings := make(map[yacymodel.Hash]yacymodel.RWIPosting, count)
	for index := range count {
		word, url := backlogPosting(index)
		postings[fakePostingKey(word, url)] = fakePosting(word, url)
	}

	return postings
}

func storeBacklog(t *testing.T, h *cycleHarness, count int) {
	t.Helper()

	for index := range count {
		word, url := backlogPosting(index)
		store(t, h.v, h.schedule, word, url)
	}
}

func backlogPosting(index int) (yacymodel.Hash, yacymodel.URLHash) {
	name := "backlog" + strconv.Itoa(index)

	return yacymodel.WordHash(name), urlHash(name)
}
