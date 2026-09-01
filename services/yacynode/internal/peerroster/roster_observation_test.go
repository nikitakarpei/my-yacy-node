package peerroster_test

import (
	"context"
	"sync"
	"testing"
)

type countingObserver struct {
	mu             sync.Mutex
	knownPeers     int
	reachablePeers int
}

func (o *countingObserver) ObserveKnownPeers(count int) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.knownPeers = count
}

func (o *countingObserver) ObserveReachablePeers(count int) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.reachablePeers = count
}

func (o *countingObserver) counts() (knownPeers, reachablePeers int) {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.knownPeers, o.reachablePeers
}

func TestObservedCountsFollowReachabilityConfirmations(t *testing.T) {
	ctx := context.Background()
	observer := &countingObserver{}
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 4, observer: observer})

	first := seniorSeed(t, "first", "203.0.113.1", 8090)
	second := seniorSeed(t, "second", "203.0.113.2", 8090)
	roster.Discover(ctx, first, second)

	if known, reachable := observer.counts(); known != 2 || reachable != 0 {
		t.Fatalf("after discovery known = %d, reachable = %d, want 2 and 0", known, reachable)
	}

	roster.ConfirmReachable(ctx, first)
	roster.ConfirmReachable(ctx, second)

	if known, reachable := observer.counts(); known != 2 || reachable != 2 {
		t.Fatalf("after confirmation known = %d, reachable = %d, want 2 and 2", known, reachable)
	}

	roster.ConfirmUnreachable(ctx, first.Hash)

	if known, reachable := observer.counts(); known != 2 || reachable != 1 {
		t.Fatalf("after failure known = %d, reachable = %d, want 2 and 1", known, reachable)
	}
}

func TestObservedReachableCountStopsAtReachableCapacity(t *testing.T) {
	ctx := context.Background()
	observer := &countingObserver{}
	roster := openRoster(t, rosterFixture{reservoirCap: 8, reachableCap: 1, observer: observer})

	admitted := seniorSeed(t, "admitted", "203.0.113.1", 8090)
	refused := seniorSeed(t, "refused", "203.0.113.2", 8090)
	roster.Discover(ctx, admitted, refused)

	roster.ConfirmReachable(ctx, admitted)
	roster.ConfirmReachable(ctx, refused)

	if _, reachable := observer.counts(); reachable != 1 {
		t.Fatalf("reachable = %d, want 1 at capacity", reachable)
	}
}
