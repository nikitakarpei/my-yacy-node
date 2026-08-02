package peerroster_test

import (
	"context"
	"testing"
	"time"
)

func TestDiscoverKeepsSeniorsAndDropsJuniors(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 4, defaultAnnounceInterval)

	senior := seniorSeed(t, "senior", "203.0.113.1", 8090)
	junior := seniorSeed(t, "junior", "", 0)
	roster.Discover(ctx, senior, junior)

	targets := hashes(roster.UnreachablePeers(ctx, 4))
	if _, ok := targets[senior.Hash]; !ok {
		t.Fatalf("senior missing from probe targets: %v", targets)
	}
	if _, ok := targets[junior.Hash]; ok {
		t.Fatalf("junior should have been dropped: %v", targets)
	}
}

func TestDiscoverDropsThisNode(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 4, defaultAnnounceInterval)

	self := seniorSeed(t, "self", "203.0.113.1", 8090)
	roster.Discover(ctx, self)
	roster.ConfirmReachable(ctx, self.Hash)

	if _, ok := hashes(roster.UnreachablePeers(ctx, 4))[self.Hash]; ok {
		t.Fatalf("this node should never be known as a peer of itself")
	}
	if got := roster.ReachablePeers(ctx); len(got) != 0 {
		t.Fatalf("reachable = %d, want 0: this node is not one of its own peers", len(got))
	}
}

func TestReachablePromotesAndIsServed(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 4, defaultAnnounceInterval)

	senior := seniorSeed(t, "senior", "203.0.113.1", 8090)
	roster.Discover(ctx, senior)

	if got := roster.ReachablePeers(ctx); len(got) != 0 {
		t.Fatalf("reachable before greet = %d, want 0", len(got))
	}

	roster.ConfirmReachable(ctx, senior.Hash)

	if _, ok := hashes(roster.ReachablePeers(ctx))[senior.Hash]; !ok {
		t.Fatalf("senior not served as reachable after confirmation")
	}
}

func TestReachableUnknownPeerIsNoop(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 4, defaultAnnounceInterval)

	roster.ConfirmReachable(ctx, hashFor("ghost"))

	if got := roster.ReachablePeers(ctx); len(got) != 0 {
		t.Fatalf("reachable = %d, want 0 for unknown peer", len(got))
	}
}

func TestUnreachableDropsFromReachableButStaysKnown(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 4, defaultAnnounceInterval)

	senior := seniorSeed(t, "senior", "203.0.113.1", 8090)
	roster.Discover(ctx, senior)
	roster.ConfirmReachable(ctx, senior.Hash)

	roster.ConfirmUnreachable(ctx, senior.Hash)

	if got := roster.ReachablePeers(ctx); len(got) != 0 {
		t.Fatalf("reachable = %d, want 0 after failure", len(got))
	}
	if _, ok := hashes(roster.UnreachablePeers(ctx, 4))[senior.Hash]; !ok {
		t.Fatalf("unreachable peer should remain known until evicted by capacity")
	}
}

func TestUnreachablePeerEvictedBeforeFresherPeers(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 2, 4, defaultAnnounceInterval)

	senior := seniorSeed(t, "senior", "203.0.113.1", 8090)
	other := seniorSeed(t, "other", "203.0.113.2", 8090)
	roster.Discover(ctx, senior)
	roster.ConfirmUnreachable(ctx, senior.Hash)
	roster.Discover(ctx, other)

	newest := seniorSeed(t, "newest", "203.0.113.3", 8090)
	roster.Discover(ctx, newest)

	targets := hashes(roster.UnreachablePeers(ctx, 4))
	if _, ok := targets[senior.Hash]; ok {
		t.Fatalf("unreachable peer should have been evicted first: %v", targets)
	}
	if len(targets) != 2 {
		t.Fatalf("reservoir size = %d, want 2 after eviction", len(targets))
	}
}

func TestDiscoverEvictsStalestBeyondCapacity(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 2, 4, defaultAnnounceInterval)

	oldest := seniorSeed(t, "oldest", "203.0.113.1", 8090)
	middle := seniorSeed(t, "middle", "203.0.113.2", 8090)
	newest := seniorSeed(t, "newest", "203.0.113.3", 8090)

	roster.Discover(ctx, oldest)
	roster.Discover(ctx, middle)
	roster.Discover(ctx, newest)

	targets := hashes(roster.UnreachablePeers(ctx, 4))
	if _, ok := targets[oldest.Hash]; ok {
		t.Fatalf("stalest peer should have been evicted: %v", targets)
	}
	if len(targets) != 2 {
		t.Fatalf("reservoir size = %d, want 2 after eviction", len(targets))
	}
}

func TestUnreachablePeersCappedToLimit(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 2, defaultAnnounceInterval)

	for _, name := range []string{"a", "b", "c", "d"} {
		roster.Discover(ctx, seniorSeed(t, name, "203.0.113.9", 8090))
	}

	if got := len(roster.UnreachablePeers(ctx, 2)); got != 2 {
		t.Fatalf("unreachable peers = %d, want capped at limit 2", got)
	}
}

func TestUnreachablePeersRotatesByLeastRecentlyContacted(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 4, defaultAnnounceInterval)

	first := seniorSeed(t, "first", "203.0.113.1", 8090)
	second := seniorSeed(t, "second", "203.0.113.2", 8090)
	roster.Discover(ctx, first)
	roster.Discover(ctx, second)

	targets := hashes(roster.UnreachablePeers(ctx, 1))
	if _, ok := targets[first.Hash]; !ok {
		t.Fatalf("least recently contacted peer missing: %v", targets)
	}

	roster.ConfirmUnreachable(ctx, first.Hash)

	targets = hashes(roster.UnreachablePeers(ctx, 1))
	if _, ok := targets[second.Hash]; !ok {
		t.Fatalf("rotation should now favor the other peer: %v", targets)
	}
}

func TestUnreachablePeersPrioritizesReachableHistoryOverNeverConfirmed(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 1, defaultAnnounceInterval)

	filler := seniorSeed(t, "filler", "203.0.113.1", 8090)
	roster.Discover(ctx, filler)
	roster.ConfirmReachable(ctx, filler.Hash)

	rejected := seniorSeed(t, "rejected", "203.0.113.2", 8090)
	roster.Discover(ctx, rejected)
	roster.ConfirmReachable(ctx, rejected.Hash)

	never := seniorSeed(t, "never", "203.0.113.3", 8090)
	roster.Discover(ctx, never)

	targets := roster.UnreachablePeers(ctx, 1)
	if len(targets) != 1 || targets[0].Hash != rejected.Hash {
		t.Fatalf(
			"probe targets = %v, want the peer confirmed reachable but rejected for capacity first",
			hashes(targets),
		)
	}
}

func TestRecentlyReachableAfterConfirmation(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 8, time.Minute)

	peer := seniorSeed(t, "peer", "203.0.113.1", 8090)
	roster.Discover(ctx, peer)
	roster.ConfirmReachable(ctx, peer.Hash)

	if !roster.IsRecentlyReachable(ctx, peer.Hash) {
		t.Fatalf("peer confirmed reachable should be recently reachable")
	}
}

func TestRecentlyReachableClearedByFailedContact(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 8, time.Minute)

	peer := seniorSeed(t, "peer", "203.0.113.1", 8090)
	roster.Discover(ctx, peer)
	roster.ConfirmReachable(ctx, peer.Hash)
	roster.ConfirmUnreachable(ctx, peer.Hash)

	if roster.IsRecentlyReachable(ctx, peer.Hash) {
		t.Fatalf("peer zeroed by ConfirmUnreachable should not be recently reachable")
	}
}

func TestRecentlyReachableExcludesUnknownPeer(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 8, time.Minute)

	if roster.IsRecentlyReachable(ctx, hashFor("ghost")) {
		t.Fatalf("unknown peer should not be recently reachable")
	}
}

func TestRecentlyReachableExcludesConfirmationPastWindow(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 8, time.Nanosecond)

	peer := seniorSeed(t, "peer", "203.0.113.1", 8090)
	roster.Discover(ctx, peer)
	roster.ConfirmReachable(ctx, peer.Hash)

	if roster.IsRecentlyReachable(ctx, peer.Hash) {
		t.Fatalf("confirmation older than the credibility window should not count")
	}
}
