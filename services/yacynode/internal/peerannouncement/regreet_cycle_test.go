package peerannouncement_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerannouncement"
)

const confirmationWait = 10 * time.Second

type stubRoster struct {
	mu               sync.Mutex
	reachablePeers   []yacymodel.Seed
	unreachablePeers []yacymodel.Seed
	discovered       []yacymodel.Seed
	reachable        []yacymodel.Hash
	unreachable      []yacymodel.Hash
	confirmations    chan struct{}
}

func newStubRoster(reachable, unreachable []yacymodel.Seed) *stubRoster {
	return &stubRoster{
		reachablePeers:   reachable,
		unreachablePeers: unreachable,
		confirmations:    make(chan struct{}, 16),
	}
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
	s.reachable = append(s.reachable, peer)
	s.mu.Unlock()

	s.confirmations <- struct{}{}
}

func (s *stubRoster) ConfirmUnreachable(_ context.Context, peer yacymodel.Hash) {
	s.mu.Lock()
	s.unreachable = append(s.unreachable, peer)
	s.mu.Unlock()

	s.confirmations <- struct{}{}
}

func (s *stubRoster) reachableHashes() []yacymodel.Hash {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]yacymodel.Hash(nil), s.reachable...)
}

func (s *stubRoster) unreachableHashes() []yacymodel.Hash {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]yacymodel.Hash(nil), s.unreachable...)
}

func (s *stubRoster) discoveredSeeds() []yacymodel.Seed {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]yacymodel.Seed(nil), s.discovered...)
}

type stubSelf struct {
	seed yacymodel.Seed
}

func (s stubSelf) SelfSeed(context.Context) yacymodel.Seed {
	return s.seed
}

type stubSeedSource struct {
	seeds []yacymodel.Seed
}

func (s *stubSeedSource) Fetch(context.Context) []yacymodel.Seed {
	return s.seeds
}

func announcerFor(
	self yacymodel.Seed,
	seeds []yacymodel.Seed,
	roster *stubRoster,
	reachableCap int,
) peerannouncement.Announcer {
	return peerannouncement.New(
		peerannouncement.Config{
			Client:             http.DefaultClient,
			NetworkName:        networkName,
			Interval:           time.Hour,
			ReachableCap:       reachableCap,
			ContactConcurrency: 4,
		},
		stubSelf{seed: self},
		&stubSeedSource{seeds: seeds},
		roster,
	)
}

func runUntilPeerConfirmed(
	t *testing.T,
	announcer peerannouncement.Announcer,
	roster *stubRoster,
) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		announcer.Run(ctx)
	}()

	select {
	case <-roster.confirmations:
	case <-time.After(confirmationWait):
		cancel()
		t.Fatal("timed out waiting for the announcer to confirm a peer")
	}

	cancel()
	<-stopped
}

func TestAnnounceRecordsReachableAndGossip(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	known := newStubPeer(t, "known", seniorAnswer(t))
	peer := newStubPeer(t, "peer", seniorAnswer(t, known.seed))

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})
	runUntilPeerConfirmed(t, announcerFor(self.seed, nil, roster, 4), roster)

	reachable := roster.reachableHashes()
	if len(reachable) != 1 || reachable[0] != peer.seed.Hash {
		t.Fatalf("reachable = %v, want [%v]", reachable, peer.seed.Hash)
	}

	discovered := roster.discoveredSeeds()
	if len(discovered) != 1 || discovered[0].Hash != known.seed.Hash {
		t.Fatalf("discovered = %v, want gossiped known seed", discovered)
	}
}

func TestAnnounceSkipsSelfInTargets(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	peer := newStubPeer(t, "peer", seniorAnswer(t))

	roster := newStubRoster(nil, []yacymodel.Seed{self.seed, peer.seed})
	runUntilPeerConfirmed(t, announcerFor(self.seed, nil, roster, 4), roster)

	if self.greetCount() != 0 {
		t.Fatalf("self greeted %d times, want 0", self.greetCount())
	}
	if peer.greetCount() != 1 {
		t.Fatalf("peer greeted %d times, want 1", peer.greetCount())
	}

	reachable := roster.reachableHashes()
	if len(reachable) != 1 || reachable[0] != peer.seed.Hash {
		t.Fatalf("reachable = %v, want [%v]", reachable, peer.seed.Hash)
	}
}

func TestAnnounceMarksFailedGreetUnreachable(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	peer := newStubPeer(t, "peer", unavailableAnswer())

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})
	runUntilPeerConfirmed(t, announcerFor(self.seed, nil, roster, 4), roster)

	unreachable := roster.unreachableHashes()
	if len(unreachable) != 1 || unreachable[0] != peer.seed.Hash {
		t.Fatalf("unreachable = %v, want [%v]", unreachable, peer.seed.Hash)
	}
	if reachable := roster.reachableHashes(); len(reachable) != 0 {
		t.Fatalf("reachable = %v, want none on failure", reachable)
	}
}

func TestAnnounceRejectsPeerThatDidNotConfirmOurNetwork(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	known := newStubPeer(t, "known", seniorAnswer(t))
	peer := newStubPeer(t, "peer", unconfirmedAnswer(t, known.seed))

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})
	runUntilPeerConfirmed(t, announcerFor(self.seed, nil, roster, 4), roster)

	unreachable := roster.unreachableHashes()
	if len(unreachable) != 1 || unreachable[0] != peer.seed.Hash {
		t.Fatalf("unreachable = %v, want [%v]", unreachable, peer.seed.Hash)
	}
	if discovered := roster.discoveredSeeds(); len(discovered) != 0 {
		t.Fatalf("discovered = %v, want no seeds from a peer outside our network", discovered)
	}
}

func TestAnnounceRefreshesReachablePeersEvenAtCapacity(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	reachablePeer := newStubPeer(t, "reachable", seniorAnswer(t))
	skippedPeer := newStubPeer(t, "skipped", seniorAnswer(t))

	roster := newStubRoster(
		[]yacymodel.Seed{reachablePeer.seed},
		[]yacymodel.Seed{skippedPeer.seed},
	)
	runUntilPeerConfirmed(t, announcerFor(self.seed, nil, roster, 1), roster)

	if reachablePeer.greetCount() != 1 {
		t.Fatalf("reachable peer greeted %d times, want 1", reachablePeer.greetCount())
	}
	if skippedPeer.greetCount() != 0 {
		t.Fatalf("peer beyond the cap greeted %d times, want 0", skippedPeer.greetCount())
	}
}

func TestRunFetchesSeedSourceOnStart(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	peer := newStubPeer(t, "peer", seniorAnswer(t))

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})
	runUntilPeerConfirmed(
		t,
		announcerFor(self.seed, []yacymodel.Seed{peer.seed}, roster, 4),
		roster,
	)

	discovered := roster.discoveredSeeds()
	if len(discovered) != 1 || discovered[0].Hash != peer.seed.Hash {
		t.Fatalf("discovered = %v, want the seed source seed on start", discovered)
	}
}
