package peerroster_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestDiscoverKeepsSeniorsAndDropsJuniors(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 4)

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

func TestReachablePromotesAndIsServed(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 4)

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
	roster := openRoster(t, 8, 4)

	roster.ConfirmReachable(ctx, hashFor("ghost"))

	if got := roster.ReachablePeers(ctx); len(got) != 0 {
		t.Fatalf("reachable = %d, want 0 for unknown peer", len(got))
	}
}

func TestUnreachableDropsFromReachableButStaysKnown(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 4)

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
	roster := openRoster(t, 2, 4)

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
	roster := openRoster(t, 2, 4)

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
	roster := openRoster(t, 8, 2)

	for _, name := range []string{"a", "b", "c", "d"} {
		roster.Discover(ctx, seniorSeed(t, name, "203.0.113.9", 8090))
	}

	if got := len(roster.UnreachablePeers(ctx, 2)); got != 2 {
		t.Fatalf("unreachable peers = %d, want capped at limit 2", got)
	}
}

func TestUnreachablePeersRotatesByLeastRecentlyContacted(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 4)

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
	roster := openRoster(t, 8, 1)

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

func TestPeersResponsibleForExcludesSelfAndNonAcceptingPeers(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 8)

	accepting := indexAcceptingSeed(t, "accepting", "203.0.113.1")
	declining := seniorSeed(t, "declining", "203.0.113.2", 8090)
	self := indexAcceptingSeed(t, "self", "203.0.113.3")

	roster.Discover(ctx, accepting, declining, self)
	roster.ConfirmReachable(ctx, accepting.Hash)
	roster.ConfirmReachable(ctx, declining.Hash)
	roster.ConfirmReachable(ctx, self.Hash)

	position, err := yacymodel.WordPosition(hashFor("word"))
	if err != nil {
		t.Fatal(err)
	}

	targets := hashes(roster.PeersResponsibleFor(ctx, position, 4))
	if _, ok := targets[accepting.Hash]; !ok {
		t.Fatalf("accepting peer missing from responsible peers: %v", targets)
	}
	if _, ok := targets[declining.Hash]; ok {
		t.Fatalf("peer without AcceptRemoteIndex should be excluded: %v", targets)
	}
	if _, ok := targets[self.Hash]; ok {
		t.Fatalf("self should be excluded from responsible peers: %v", targets)
	}
}

func TestPeersResponsibleForOrdersByRingDistanceAndCapsWant(t *testing.T) {
	ctx := context.Background()
	roster := openRoster(t, 8, 8)

	near := indexAcceptingSeed(t, "near", "203.0.113.1")
	far := indexAcceptingSeed(t, "far", "203.0.113.2")

	roster.Discover(ctx, near, far)
	roster.ConfirmReachable(ctx, near.Hash)
	roster.ConfirmReachable(ctx, far.Hash)

	nearPos, err := yacymodel.WordPosition(near.Hash)
	if err != nil {
		t.Fatal(err)
	}

	targets := roster.PeersResponsibleFor(ctx, nearPos, 1)
	if len(targets) != 1 {
		t.Fatalf("responsible peers = %d, want 1", len(targets))
	}
	if targets[0].Hash != near.Hash {
		t.Fatalf(
			"responsible peer = %v, want the peer closest to the position (%v)",
			targets[0].Hash, near.Hash,
		)
	}
}
