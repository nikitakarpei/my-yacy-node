package peerannouncement

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type stubGreeter struct {
	result greetResult
	err    error
	calls  int
}

func (g *stubGreeter) Greet(context.Context, string, yacymodel.Seed, int) (greetResult, error) {
	g.calls++

	return g.result, g.err
}

type stubRoster struct {
	mu               sync.Mutex
	reachablePeers   []yacymodel.Seed
	unreachablePeers []yacymodel.Seed
	discovered       []yacymodel.Seed
	reachable        []yacymodel.Hash
	unreachable      []yacymodel.Hash
}

func (s *stubRoster) ReachablePeers(context.Context) []yacymodel.Seed {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]yacymodel.Seed(nil), s.reachablePeers...)
}

func (s *stubRoster) UnreachablePeers(_ context.Context, limit int) []yacymodel.Seed {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit > len(s.unreachablePeers) {
		limit = len(s.unreachablePeers)
	}

	return append([]yacymodel.Seed(nil), s.unreachablePeers[:limit]...)
}

func (s *stubRoster) Discover(_ context.Context, seeds ...yacymodel.Seed) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.discovered = append(s.discovered, seeds...)
}

func (s *stubRoster) ConfirmReachable(_ context.Context, peer yacymodel.Hash) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.reachable = append(s.reachable, peer)
}

func (s *stubRoster) ConfirmUnreachable(_ context.Context, peer yacymodel.Hash) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.unreachable = append(s.unreachable, peer)
}

type stubSelf struct {
	seed yacymodel.Seed
}

func (s stubSelf) SelfSeed(context.Context) yacymodel.Seed {
	return s.seed
}

type stubSeedSource struct {
	seeds []yacymodel.Seed
	calls int
}

func (s *stubSeedSource) Fetch(context.Context) []yacymodel.Seed {
	s.calls++

	return s.seeds
}

func TestAnnounceRecordsReachableAndGossip(t *testing.T) {
	ctx := context.Background()
	peer := callerSeed(t, "peer", "203.0.113.1")
	known := callerSeed(t, "known", "198.51.100.7")

	roster := &stubRoster{unreachablePeers: []yacymodel.Seed{peer}}
	a := &announcer{
		reachableCap:       4,
		contactConcurrency: 4,
		self:               stubSelf{seed: callerSeed(t, "self", "203.0.113.9")},
		seeds:              &stubSeedSource{},
		roster:             roster,
		greeter: &stubGreeter{result: greetResult{
			YourType: yacymodel.Some(yacymodel.PeerSenior),
			Known:    []yacymodel.Seed{known},
		}},
	}

	a.Announce(ctx)

	if len(roster.reachable) != 1 || roster.reachable[0] != peer.Hash {
		t.Fatalf("reachable = %v, want [%v]", roster.reachable, peer.Hash)
	}
	if len(roster.discovered) != 1 || roster.discovered[0].Hash != known.Hash {
		t.Fatalf("discovered = %v, want gossiped known seed", roster.discovered)
	}
}

func TestAnnounceSkipsSelfInTargets(t *testing.T) {
	ctx := context.Background()
	self := callerSeed(t, "self", "203.0.113.9")

	roster := &stubRoster{unreachablePeers: []yacymodel.Seed{self}}
	greeter := &stubGreeter{result: greetResult{YourType: yacymodel.Some(yacymodel.PeerSenior)}}
	a := &announcer{
		reachableCap:       4,
		contactConcurrency: 4,
		self:               stubSelf{seed: self},
		seeds:              &stubSeedSource{},
		roster:             roster,
		greeter:            greeter,
	}

	a.Announce(ctx)

	if greeter.calls != 0 {
		t.Fatalf("greeter calls = %d, want 0 when only self is a target", greeter.calls)
	}
	if len(roster.reachable) != 0 {
		t.Fatalf("reachable = %v, want none for self", roster.reachable)
	}
}

func TestAnnounceMarksFailedGreetUnreachable(t *testing.T) {
	ctx := context.Background()
	peer := callerSeed(t, "peer", "203.0.113.1")

	roster := &stubRoster{unreachablePeers: []yacymodel.Seed{peer}}
	a := &announcer{
		reachableCap:       4,
		contactConcurrency: 4,
		self:               stubSelf{seed: callerSeed(t, "self", "203.0.113.9")},
		seeds:              &stubSeedSource{},
		roster:             roster,
		greeter:            &stubGreeter{err: errors.New("boom")},
	}

	a.Announce(ctx)

	if len(roster.unreachable) != 1 || roster.unreachable[0] != peer.Hash {
		t.Fatalf("unreachable = %v, want [%v]", roster.unreachable, peer.Hash)
	}
	if len(roster.reachable) != 0 {
		t.Fatalf("reachable = %v, want none on failure", roster.reachable)
	}
}

func TestAnnounceRejectsPeerThatDidNotConfirmOurNetwork(t *testing.T) {
	ctx := context.Background()
	peer := callerSeed(t, "peer", "203.0.113.1")
	known := callerSeed(t, "known", "198.51.100.7")

	roster := &stubRoster{unreachablePeers: []yacymodel.Seed{peer}}
	a := &announcer{
		reachableCap:       4,
		contactConcurrency: 4,
		self:               stubSelf{seed: callerSeed(t, "self", "203.0.113.9")},
		seeds:              &stubSeedSource{},
		roster:             roster,
		greeter: &stubGreeter{result: greetResult{
			Known: []yacymodel.Seed{known},
		}},
	}

	a.Announce(ctx)

	if len(roster.unreachable) != 1 || roster.unreachable[0] != peer.Hash {
		t.Fatalf("unreachable = %v, want [%v]", roster.unreachable, peer.Hash)
	}
	if len(roster.reachable) != 0 {
		t.Fatalf("reachable = %v, want none for a peer outside our network", roster.reachable)
	}
	if len(roster.discovered) != 0 {
		t.Fatalf(
			"discovered = %v, want no seeds gossiped by a peer outside our network",
			roster.discovered,
		)
	}
}

func TestAnnounceRefreshesReachablePeersEvenAtCapacity(t *testing.T) {
	ctx := context.Background()
	reachablePeer := callerSeed(t, "reachable", "203.0.113.1")
	skippedPeer := callerSeed(t, "skipped", "203.0.113.2")

	roster := &stubRoster{
		reachablePeers:   []yacymodel.Seed{reachablePeer},
		unreachablePeers: []yacymodel.Seed{skippedPeer},
	}
	greeter := &stubGreeter{result: greetResult{YourType: yacymodel.Some(yacymodel.PeerSenior)}}
	a := &announcer{
		reachableCap:       1,
		contactConcurrency: 4,
		self:               stubSelf{seed: callerSeed(t, "self", "203.0.113.9")},
		seeds:              &stubSeedSource{},
		roster:             roster,
		greeter:            greeter,
	}

	a.Announce(ctx)

	if greeter.calls != 1 {
		t.Fatalf("greeter calls = %d, want 1 for the reachable peer only", greeter.calls)
	}
	if len(roster.reachable) != 1 || roster.reachable[0] != reachablePeer.Hash {
		t.Fatalf("reachable = %v, want refresh of [%v]", roster.reachable, reachablePeer.Hash)
	}
}

func TestRunFetchesSeedSourceOnStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	seed := callerSeed(t, "seed", "203.0.113.1")
	source := &stubSeedSource{seeds: []yacymodel.Seed{seed}}
	roster := &stubRoster{unreachablePeers: []yacymodel.Seed{seed}}
	a := &announcer{
		interval:           time.Hour,
		reachableCap:       4,
		contactConcurrency: 4,
		self:               stubSelf{seed: callerSeed(t, "self", "203.0.113.9")},
		seeds:              source,
		roster:             roster,
		greeter: &stubGreeter{
			result: greetResult{YourType: yacymodel.Some(yacymodel.PeerSenior)},
		},
	}

	a.Run(ctx)

	if source.calls != 1 {
		t.Fatalf("seed source calls = %d, want 1 on start", source.calls)
	}
}
