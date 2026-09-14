package peerroster_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestDiscoverKeepsSeniorsAndDropsJuniors(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})

	senior := seniorSeed(t, "senior", "203.0.113.1", 8090)
	junior := seniorSeed(t, "junior", "", 0)
	roster.Discover(ctx, senior, junior)

	targets := hashSet(roster.UnreachablePeerHashes(ctx, 4))
	if _, ok := targets[senior.Hash]; !ok {
		t.Fatalf("senior missing from probe targets: %v", targets)
	}
	if _, ok := targets[junior.Hash]; ok {
		t.Fatalf("junior should have been dropped: %v", targets)
	}
}

func TestDiscoverDropsThisNode(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})

	self := seniorSeed(t, "self", "203.0.113.1", 8090)
	roster.Discover(ctx, self)
	roster.ConfirmReachable(ctx, self)

	if _, ok := hashSet(roster.UnreachablePeerHashes(ctx, 4))[self.Hash]; ok {
		t.Fatalf("this node should never be known as a peer of itself")
	}
	if got := roster.ReachablePeers(ctx); len(got) != 0 {
		t.Fatalf("reachable = %d, want 0: this node is not one of its own peers", len(got))
	}
}

func TestReachablePromotesAndIsServed(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})

	senior := seniorSeed(t, "senior", "203.0.113.1", 8090)
	roster.Discover(ctx, senior)

	if got := roster.ReachablePeers(ctx); len(got) != 0 {
		t.Fatalf("reachable before greet = %d, want 0", len(got))
	}

	roster.ConfirmReachable(ctx, senior)

	if _, ok := hashes(roster.ReachablePeers(ctx))[senior.Hash]; !ok {
		t.Fatalf("senior not served as reachable after confirmation")
	}
}

func TestReachableConfirmationReplacesDiscoveredSeedAndNetworkAddress(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})

	discovered := seniorSeed(t, "peer", "203.0.113.1", 8090)
	roster.Discover(ctx, discovered)
	fresh := seniorSeed(t, "peer", "203.0.113.2", 8091)
	fresh.Capabilities = yacymodel.Some(yacymodel.PeerCapabilities{AcceptRemoteIndex: true})
	roster.ConfirmReachable(ctx, fresh)

	reachablePeers := roster.ReachablePeers(ctx)
	if len(reachablePeers) != 1 || reachablePeers[0].Hash != fresh.Hash {
		t.Fatalf("reachable peers = %v, want fresh peer", hashes(reachablePeers))
	}
	capabilities, present := reachablePeers[0].Capabilities.Get()
	if !present || !capabilities.AcceptRemoteIndex {
		t.Fatalf("reachable peer did not retain fresh capabilities")
	}
	address, found := roster.NetworkAddressOf(ctx, fresh.Hash)
	if !found {
		t.Fatal("fresh network address not found")
	}
	wantAddress, _ := fresh.NetworkAddress()
	if address != wantAddress {
		t.Errorf("network address = %v, want %v", address, wantAddress)
	}
}

func TestReachableUnknownPeerIsAdmitted(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})
	peer := seniorSeed(t, "ghost", "203.0.113.8", 8090)

	roster.ConfirmReachable(ctx, peer)

	if _, admitted := hashes(roster.ReachablePeers(ctx))[peer.Hash]; !admitted {
		t.Fatalf("confirmed unknown peer was not admitted")
	}
}

func TestUnreachableDropsFromReachableButStaysKnown(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})

	senior := seniorSeed(t, "senior", "203.0.113.1", 8090)
	roster.Discover(ctx, senior)
	roster.ConfirmReachable(ctx, senior)

	roster.ConfirmUnreachable(ctx, senior.Hash)

	if got := roster.ReachablePeers(ctx); len(got) != 0 {
		t.Fatalf("reachable = %d, want 0 after failure", len(got))
	}
	if _, ok := hashSet(roster.UnreachablePeerHashes(ctx, 4))[senior.Hash]; !ok {
		t.Fatalf("unreachable peer should remain known until evicted by capacity")
	}
}

func TestUnreachablePeerEvictedBeforeFresherPeers(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 2, reachableCap: 4})

	senior := seniorSeed(t, "senior", "203.0.113.1", 8090)
	other := seniorSeed(t, "other", "203.0.113.2", 8090)
	roster.Discover(ctx, senior)
	roster.ConfirmUnreachable(ctx, senior.Hash)
	roster.Discover(ctx, other)

	newest := seniorSeed(t, "newest", "203.0.113.3", 8090)
	roster.Discover(ctx, newest)

	targets := hashSet(roster.UnreachablePeerHashes(ctx, 4))
	if _, ok := targets[senior.Hash]; ok {
		t.Fatalf("unreachable peer should have been evicted first: %v", targets)
	}
	if len(targets) != 2 {
		t.Fatalf("reservoir size = %d, want 2 after eviction", len(targets))
	}
}

func TestDiscoverEvictsStalestBeyondCapacity(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 2, reachableCap: 4})

	oldest := seniorSeed(t, "oldest", "203.0.113.1", 8090)
	middle := seniorSeed(t, "middle", "203.0.113.2", 8090)
	newest := seniorSeed(t, "newest", "203.0.113.3", 8090)

	roster.Discover(ctx, oldest)
	roster.Discover(ctx, middle)
	roster.Discover(ctx, newest)

	targets := hashSet(roster.UnreachablePeerHashes(ctx, 4))
	if _, ok := targets[oldest.Hash]; ok {
		t.Fatalf("stalest peer should have been evicted: %v", targets)
	}
	if len(targets) != 2 {
		t.Fatalf("reservoir size = %d, want 2 after eviction", len(targets))
	}
}

func TestMostRecentlyReachablePeersRankLatestConfirmationFirst(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})

	earlier := seniorSeed(t, "earlier", "203.0.113.1", 8090)
	later := seniorSeed(t, "later", "203.0.113.2", 8090)
	roster.Discover(ctx, earlier, later)
	roster.ConfirmReachable(ctx, earlier)
	roster.ConfirmReachable(ctx, later)

	ranked := roster.MostRecentlyReachablePeers(ctx, 4)
	if len(ranked) != 2 || ranked[0].Hash != later.Hash {
		t.Fatalf("ranked = %v, want the last confirmed peer first", hashes(ranked))
	}

	roster.ConfirmReachable(ctx, earlier)

	ranked = roster.MostRecentlyReachablePeers(ctx, 4)
	if len(ranked) != 2 || ranked[0].Hash != earlier.Hash {
		t.Fatalf("ranked = %v, want the reconfirmed peer first", hashes(ranked))
	}
}

func TestMostRecentlyReachablePeersCappedToLimit(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})

	first := seniorSeed(t, "first", "203.0.113.1", 8090)
	second := seniorSeed(t, "second", "203.0.113.2", 8090)
	roster.Discover(ctx, first, second)
	roster.ConfirmReachable(ctx, first)
	roster.ConfirmReachable(ctx, second)

	if got := roster.MostRecentlyReachablePeers(ctx, 1); len(got) != 1 {
		t.Fatalf("ranked = %d, want 1", len(got))
	}
	if got := roster.MostRecentlyReachablePeers(ctx, 0); len(got) != 0 {
		t.Fatalf("ranked = %d, want none for a limit of zero", len(got))
	}
}

func TestUnreachablePeerHashesCappedToLimit(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 2})

	for _, name := range []string{"a", "b", "c", "d"} {
		roster.Discover(ctx, seniorSeed(t, name, "203.0.113.9", 8090))
	}

	if got := len(roster.UnreachablePeerHashes(ctx, 2)); got != 2 {
		t.Fatalf("unreachable peers = %d, want capped at limit 2", got)
	}
}

func TestUnreachablePeerHashesRotateByLeastRecentlyContacted(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4})

	first := seniorSeed(t, "first", "203.0.113.1", 8090)
	second := seniorSeed(t, "second", "203.0.113.2", 8090)
	roster.Discover(ctx, first)
	roster.Discover(ctx, second)

	targets := hashSet(roster.UnreachablePeerHashes(ctx, 1))
	if _, ok := targets[first.Hash]; !ok {
		t.Fatalf("least recently contacted peer missing: %v", targets)
	}

	roster.ConfirmUnreachable(ctx, first.Hash)

	targets = hashSet(roster.UnreachablePeerHashes(ctx, 1))
	if _, ok := targets[second.Hash]; !ok {
		t.Fatalf("rotation should now favor the other peer: %v", targets)
	}
}

func TestUnreachablePeerHashesPrioritizeReachableHistoryOverNeverConfirmed(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 1})

	filler := seniorSeed(t, "filler", "203.0.113.1", 8090)
	roster.Discover(ctx, filler)
	roster.ConfirmReachable(ctx, filler)

	rejected := seniorSeed(t, "rejected", "203.0.113.2", 8090)
	roster.Discover(ctx, rejected)
	roster.ConfirmReachable(ctx, rejected)

	never := seniorSeed(t, "never", "203.0.113.3", 8090)
	roster.Discover(ctx, never)

	targets := roster.UnreachablePeerHashes(ctx, 1)
	if len(targets) != 1 || targets[0] != rejected.Hash {
		t.Fatalf(
			"probe targets = %v, want the peer confirmed reachable but rejected for capacity first",
			hashSet(targets),
		)
	}
}

func TestRecentlyReachableAfterConfirmation(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(
		t,
		rosterFixture{reservoirCap: 8, reachableCap: 8, announceInterval: time.Minute},
	)

	peer := seniorSeed(t, "peer", "203.0.113.1", 8090)
	roster.Discover(ctx, peer)
	roster.ConfirmReachable(ctx, peer)

	if !roster.IsRecentlyReachable(ctx, peer.Hash) {
		t.Fatalf("peer confirmed reachable should be recently reachable")
	}
}

func TestRecentlyReachableClearedByFailedContact(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(
		t,
		rosterFixture{reservoirCap: 8, reachableCap: 8, announceInterval: time.Minute},
	)

	peer := seniorSeed(t, "peer", "203.0.113.1", 8090)
	roster.Discover(ctx, peer)
	roster.ConfirmReachable(ctx, peer)
	roster.ConfirmUnreachable(ctx, peer.Hash)

	if roster.IsRecentlyReachable(ctx, peer.Hash) {
		t.Fatalf("peer zeroed by ConfirmUnreachable should not be recently reachable")
	}
}

func TestRecentlyReachableExcludesUnknownPeer(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(
		t,
		rosterFixture{reservoirCap: 8, reachableCap: 8, announceInterval: time.Minute},
	)

	if roster.IsRecentlyReachable(ctx, hashFor("ghost")) {
		t.Fatalf("unknown peer should not be recently reachable")
	}
}

func TestRecentlyReachableExcludesConfirmationPastWindow(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(
		t,
		rosterFixture{reservoirCap: 8, reachableCap: 8, announceInterval: time.Nanosecond},
	)

	peer := seniorSeed(t, "peer", "203.0.113.1", 8090)
	roster.Discover(ctx, peer)
	roster.ConfirmReachable(ctx, peer)

	if roster.IsRecentlyReachable(ctx, peer.Hash) {
		t.Fatalf("confirmation older than the credibility window should not count")
	}
}
