package distributioncycle

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/urlmetadatacourier"
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
	h.courier.receipts[peer] = postingcourier.PostingReceipt{Outcome: postingcourier.Accepted}

	store(t, h.v, h.schedule, word, url)

	ctx, cancel := context.WithCancel(context.Background())
	h.courier.onOffer = cancel
	h.cycle.Run(ctx)

	if len(h.courier.offered) != 1 {
		t.Fatalf("offered = %v, want a single offer from the initial run", h.courier.offered)
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
		postings: postings,
		roster:   fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
	})
	h.cycle.minReachablePeers = 2

	store(t, h.v, h.schedule, word, url)

	h.cycle.runCycle(context.Background())

	if len(h.courier.offered) != 0 {
		t.Fatalf(
			"offered = %v, want no offers while below the reachable-peer floor",
			h.courier.offered,
		)
	}
	if h.observer.cyclesSkipped[skipTooFewReachablePeers] != 1 {
		t.Fatalf("cyclesSkipped = %v, want one skip below the reachable-peer floor",
			h.observer.cyclesSkipped)
	}

	due, err := h.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] left untouched by a skipped cycle", due)
	}
}

func TestCycleReportsBacklogAgeOnSkippedCycle(t *testing.T) {
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
	h.cycle.minReachablePeers = 2

	store(t, h.v, h.schedule, word, url)

	clk.at = now.Add(90 * time.Second)
	h.cycle.runCycle(context.Background())

	if len(h.courier.offered) != 0 {
		t.Fatalf("offered = %v, want no offers on a skipped cycle", h.courier.offered)
	}
	if h.observer.longestOfferLateness != 90*time.Second {
		t.Fatalf("longestOfferLateness = %v, want 90s", h.observer.longestOfferLateness)
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
	h.courier.receipts[fresh] = postingcourier.PostingReceipt{Outcome: postingcourier.Accepted}

	store(t, h.v, h.schedule, word, url)
	recordAccepted(t, h, stalePeer, fakePosting(word, url))

	h.cycle.runCycle(context.Background())

	replicas, err := h.replicas.Holders(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Holders: %v", err)
	}
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

	h.cycle.runCycle(context.Background())

	replicas, err := h.replicas.Holders(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Holders: %v", err)
	}
	if len(replicas) != 1 || replicas[0] != recentPeer {
		t.Fatalf(
			"replicas = %v, want %v kept while its confirmation is still credible",
			replicas, recentPeer,
		)
	}
	if h.observer.staleReplicasDropped != 0 {
		t.Fatalf("staleReplicasDropped = %v, want 0", h.observer.staleReplicasDropped)
	}
	if len(h.courier.offered) != 0 {
		t.Fatalf(
			"offered = %v, want no offer once the credible peer already holds a replica",
			h.courier.offered,
		)
	}
}

func TestCycleReschedulesAcceptedPostingAtRefreshInterval(t *testing.T) {
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
	h.courier.receipts[peer] = postingcourier.PostingReceipt{Outcome: postingcourier.Accepted}

	store(t, h.v, h.schedule, word, url)

	h.cycle.runCycle(context.Background())

	due, err := h.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
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
	h.courier.receipts[peer] = postingcourier.PostingReceipt{Outcome: postingcourier.Refused}

	store(t, h.v, h.schedule, word, url)

	h.cycle.runCycle(context.Background())

	due, err := h.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due immediately after a rejected offer", due)
	}

	clk.at = now.Add(h.bounds.First + time.Second)
	due, err = h.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings after retry interval: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] once the retry interval has elapsed", due)
	}
}

func TestCycleHonoursCourierRetryAfter(t *testing.T) {
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
	h.courier.receipts[peer] = postingcourier.PostingReceipt{
		Outcome:    postingcourier.Deferred,
		RetryAfter: 5 * time.Minute,
	}

	store(t, h.v, h.schedule, word, url)

	h.cycle.runCycle(context.Background())

	clk.at = now.Add(h.bounds.First + time.Second)
	due, err := h.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due before the peer's pause elapses", due)
	}

	clk.at = now.Add(5*time.Minute + time.Second)
	due, err = h.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings after pause: %v", err)
	}
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
	h := openCycle(t, clk, cycleOptions{postings: postings})

	store(t, h.v, h.schedule, word, url)

	h.cycle.runCycle(context.Background())

	if len(h.courier.offered) != 0 {
		t.Fatalf("offered = %v, want no offers for an unoffered posting", h.courier.offered)
	}

	due, err := h.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due immediately after stalling", due)
	}

	clk.at = now.Add(h.bounds.First + time.Second)
	due, err = h.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings after retry interval: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] once the retry interval has elapsed", due)
	}
}

func TestCycleReschedulesAlreadySatisfiedPostingAtRefreshInterval(t *testing.T) {
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

	store(t, h.v, h.schedule, word, url)
	recordAccepted(t, h, peer, fakePosting(word, url))

	h.cycle.runCycle(context.Background())

	if len(h.courier.offered) != 1 || h.courier.offered[0].Peer.Hash != peer {
		t.Fatalf(
			"offered = %v, want the posting renewed with the peer that holds it",
			h.courier.offered,
		)
	}

	clk.at = now.Add(h.bounds.Longest - time.Second)
	due, err := h.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due before the refresh interval elapses", due)
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
		postings: postings,
		roster:   fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
	})
	h.courier.receipts[peer] = postingcourier.PostingReceipt{Outcome: postingcourier.Accepted}

	store(t, h.v, h.schedule, word, url)

	h.cycle.runCycle(context.Background())

	replicas, err := h.replicas.Holders(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Holders: %v", err)
	}
	if len(replicas) != 1 || replicas[0] != peer {
		t.Fatalf("replicas = %v, want [%v]", replicas, peer)
	}
	if h.observer.postingsOffered[string(postingcourier.Accepted)] != 1 {
		t.Fatalf(
			"observed offers = %+v, want 1 posting for outcome %q",
			h.observer.postingsOffered, postingcourier.Accepted,
		)
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
	h.courier.receipts[peerA] = postingcourier.PostingReceipt{
		Outcome:    postingcourier.Deferred,
		RetryAfter: time.Minute,
	}
	h.courier.receipts[peerB] = postingcourier.PostingReceipt{
		Outcome:    postingcourier.Deferred,
		RetryAfter: 5 * time.Minute,
	}

	store(t, h.v, h.schedule, word, url)

	h.cycle.runCycle(context.Background())

	clk.at = now.Add(time.Minute + time.Second)
	due, err := h.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due before the longer pause elapses", due)
	}

	clk.at = now.Add(5*time.Minute + time.Second)
	due, err = h.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings after longer pause: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] once the longer pause has elapsed", due)
	}
}

func TestCycleCountsPostingGoneFromIndex(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	h := openCycle(t, &clock{at: now}, cycleOptions{
		roster: fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
	})

	store(t, h.v, h.schedule, word, url)

	h.cycle.runCycle(context.Background())

	if h.observer.gone != 1 {
		t.Fatalf("gone = %v, want 1", h.observer.gone)
	}
	if len(h.courier.offered) != 0 {
		t.Fatalf(
			"offered = %v, want no offers for a posting gone from the index",
			h.courier.offered,
		)
	}
}

func TestCycleCountsUnreadShortfall(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	h := openCycle(t, &clock{at: now}, cycleOptions{
		roster:      fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
		postingsErr: errors.New("vault closed"),
	})

	store(t, h.v, h.schedule, word, url)

	h.cycle.runCycle(context.Background())

	if h.observer.cyclesSkipped[skipShortfallUnread] != 1 {
		t.Fatalf("cyclesSkipped = %v, want one skip for an unread shortfall",
			h.observer.cyclesSkipped)
	}
	if len(h.courier.offered) != 0 {
		t.Fatalf("offered = %v, want no offers when the replication is unread", h.courier.offered)
	}
}

func TestCycleReportsAnEmptyScheduleAsNoLateness(t *testing.T) {
	now := time.Unix(1000, 0)
	peer := yacymodel.WordHash("peer")
	h := openCycle(t, &clock{at: now}, cycleOptions{
		roster: fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
	})
	h.observer.longestOfferLateness = time.Hour

	h.cycle.runCycle(context.Background())

	if h.observer.longestOfferLateness != 0 {
		t.Fatalf(
			"longestOfferLateness = %v, want 0 while nothing is scheduled",
			h.observer.longestOfferLateness,
		)
	}
	if h.observer.scheduledPostings != 0 {
		t.Fatalf(
			"scheduledPostings = %d, want 0 while nothing is scheduled",
			h.observer.scheduledPostings,
		)
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

	h.cycle.runCycle(context.Background())

	if h.observer.scheduledPostings != 1 {
		t.Fatalf(
			"scheduledPostings = %d, want 1 for the single stored posting",
			h.observer.scheduledPostings,
		)
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
	h.courier.receipts[peer] = postingcourier.PostingReceipt{Outcome: postingcourier.Refused}

	store(t, h.v, h.schedule, word, url)

	h.cycle.runCycle(context.Background())

	replicas, err := h.replicas.Holders(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Holders: %v", err)
	}
	if len(replicas) != 0 {
		t.Fatalf("replicas = %v, want none after a refused offer", replicas)
	}
	if h.observer.postingsOffered[string(postingcourier.Refused)] != 1 {
		t.Fatalf(
			"observed offers = %+v, want 1 posting for outcome %q",
			h.observer.postingsOffered, postingcourier.Refused,
		)
	}
}

func TestCycleReportsUnaddressablePeer(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	unaddressable := yacymodel.Seed{
		Hash:         peer,
		Capabilities: yacymodel.Some(yacymodel.PeerCapabilities{AcceptRemoteIndex: true}),
	}
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings: postings,
		roster:   fakeRoster{reachable: []yacymodel.Seed{unaddressable}},
	})

	store(t, h.v, h.schedule, word, url)

	h.cycle.runCycle(context.Background())

	if len(h.courier.offered) != 0 {
		t.Fatalf("offered = %v, want no offer sent to an unaddressable peer", h.courier.offered)
	}
	if h.observer.postingsOffered[string(postingcourier.Unaddressable)] != 1 {
		t.Fatalf(
			"observed offers = %+v, want 1 posting for outcome %q",
			h.observer.postingsOffered, postingcourier.Unaddressable,
		)
	}
	replicas, err := h.replicas.Holders(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Holders: %v", err)
	}
	if len(replicas) != 0 {
		t.Fatalf("replicas = %v, want none for an unaddressable peer", replicas)
	}
}

func TestCycleExcludesPostingWhenURLMetadataDeliveryFails(t *testing.T) {
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
		urls: fakeURLDirectory{
			metadata: map[yacymodel.URLHash]yacymodel.URLMetadata{
				url: {Address: "http://example.com/u1"},
			},
		},
		metadataOutcome: urlmetadatacourier.Deferred,
	})
	h.courier.receipts[peer] = postingcourier.PostingReceipt{
		Outcome:           postingcourier.Accepted,
		URLsUnknownToPeer: []yacymodel.URLHash{url},
	}

	store(t, h.v, h.schedule, word, url)

	h.cycle.runCycle(context.Background())

	replicas, err := h.replicas.Holders(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Holders: %v", err)
	}
	if len(replicas) != 0 {
		t.Fatalf("replicas = %v, want none recorded when url metadata delivery fails", replicas)
	}
	if h.observer.urlMetadataDeliveries[string(urlmetadatacourier.Deferred)] != 1 {
		t.Fatalf(
			"observed url metadata deliveries = %+v, want 1 for outcome %q",
			h.observer.urlMetadataDeliveries, urlmetadatacourier.Deferred,
		)
	}

	clk.at = now.Add(h.bounds.First + time.Second)
	due, err := h.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf(
			"due = %v, want [word] retried after a failed url metadata delivery",
			due,
		)
	}
}

func TestCycleDeliversMetadataItHasWhenOneURLIsAbsent(t *testing.T) {
	now := time.Unix(1000, 0)
	word := yacymodel.WordHash("w1")
	present, absent := urlHash("u1"), urlHash("u2")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, present): fakePosting(word, present),
		fakePostingKey(word, absent):  fakePosting(word, absent),
	}
	h := openCycle(t, &clock{at: now}, cycleOptions{
		postings: postings,
		roster:   fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
		urls: fakeURLDirectory{
			metadata: map[yacymodel.URLHash]yacymodel.URLMetadata{
				present: {Address: "http://example.com/u1"},
			},
		},
	})
	h.courier.receipts[peer] = postingcourier.PostingReceipt{
		Outcome:           postingcourier.Accepted,
		URLsUnknownToPeer: []yacymodel.URLHash{present, absent},
	}

	store(t, h.v, h.schedule, word, present)
	store(t, h.v, h.schedule, word, absent)

	h.cycle.runCycle(context.Background())

	if len(h.metadataCourier.delivered) != 1 {
		t.Fatalf(
			"delivered = %v, want the one url whose metadata this node holds",
			h.metadataCourier.delivered,
		)
	}
	if h.observer.urlMetadataDeliveries[string(urlmetadatacourier.Unavailable)] != 1 {
		t.Fatalf(
			"observed url metadata deliveries = %+v, want 1 for outcome %q",
			h.observer.urlMetadataDeliveries, urlmetadatacourier.Unavailable,
		)
	}

	replicas, err := h.replicas.Holders(context.Background(), word, present)
	if err != nil {
		t.Fatalf("Holders: %v", err)
	}
	if len(replicas) != 1 {
		t.Fatalf("replicas = %v, want the deliverable posting recorded", replicas)
	}

	replicas, err = h.replicas.Holders(context.Background(), word, absent)
	if err != nil {
		t.Fatalf("Holders: %v", err)
	}
	if len(replicas) != 0 {
		t.Fatalf("replicas = %v, want none for the posting with no url metadata", replicas)
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
		h.courier.receipts[peer.Hash] = postingcourier.PostingReceipt{
			Outcome: postingcourier.Unreachable,
		}
	}

	store(t, h.v, h.schedule, word, url)

	h.cycle.runCycle(context.Background())
	clk.at = clk.at.Add(2 * h.bounds.First)
	h.cycle.runCycle(context.Background())

	if len(h.courier.offered) != 2 {
		t.Fatalf("offered = %v, want one offer from each cycle", h.courier.offered)
	}
	if h.courier.offered[0].Peer.Hash == h.courier.offered[1].Peer.Hash {
		t.Fatalf(
			"both offers went to %v, want the second cycle to try another peer",
			h.courier.offered[0].Peer.Hash,
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
	h := openCycle(t, clk, cycleOptions{postings: postings})

	store(t, h.v, h.schedule, word, url)

	h.cycle.runCycle(context.Background())
	clk.at = now.Add(h.bounds.First)
	h.cycle.runCycle(context.Background())

	clk.at = now.Add(2*h.bounds.First + time.Second)
	due, err := h.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none: the second miss doubles the wait", due)
	}

	clk.at = now.Add(3*h.bounds.First + time.Second)
	due, err = h.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings after the doubled wait: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] once the doubled wait has elapsed", due)
	}
}
