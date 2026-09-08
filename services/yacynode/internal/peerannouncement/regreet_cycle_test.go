package peerannouncement_test

import (
	"context"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerannouncement"
)

const confirmationWait = 10 * time.Second

type stubRoster struct {
	mu                     sync.Mutex
	reachablePeers         []yacymodel.Seed
	unreachablePeerHashes  []yacymodel.Hash
	networkAddresses       map[yacymodel.Hash]yacymodel.NetworkAddress
	discovered             []yacymodel.Seed
	reachable              []yacymodel.Hash
	unreachable            []yacymodel.Hash
	reachableConfirmations []yacymodel.Seed
	confirmations          chan struct{}
}

func newStubRoster(reachable, unreachable []yacymodel.Seed) *stubRoster {
	networkAddresses := make(map[yacymodel.Hash]yacymodel.NetworkAddress)
	for _, seed := range append(append([]yacymodel.Seed{}, reachable...), unreachable...) {
		networkAddress, addressable := seed.NetworkAddress()
		if addressable {
			networkAddresses[seed.Hash] = networkAddress
		}
	}
	unreachablePeerHashes := make([]yacymodel.Hash, len(unreachable))
	for index, seed := range unreachable {
		unreachablePeerHashes[index] = seed.Hash
	}

	return &stubRoster{
		reachablePeers:        reachable,
		unreachablePeerHashes: unreachablePeerHashes,
		networkAddresses:      networkAddresses,
		confirmations:         make(chan struct{}, 16),
	}
}

func (s *stubRoster) ReachablePeers(context.Context) []yacymodel.Seed {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]yacymodel.Seed(nil), s.reachablePeers...)
}

func (s *stubRoster) UnreachablePeerHashes(_ context.Context, limit int) []yacymodel.Hash {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit > len(s.unreachablePeerHashes) {
		limit = len(s.unreachablePeerHashes)
	}

	return append([]yacymodel.Hash(nil), s.unreachablePeerHashes[:limit]...)
}

func (s *stubRoster) NetworkAddressOf(
	_ context.Context,
	peer yacymodel.Hash,
) (yacymodel.NetworkAddress, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	networkAddress, found := s.networkAddresses[peer]

	return networkAddress, found
}

func (s *stubRoster) Discover(_ context.Context, seeds ...yacymodel.Seed) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.discovered = append(s.discovered, seeds...)
}

func (s *stubRoster) ConfirmReachable(_ context.Context, seed yacymodel.Seed) {
	s.mu.Lock()
	s.reachable = append(s.reachable, seed.Hash)
	s.reachableConfirmations = append(s.reachableConfirmations, seed)
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

func (s *stubRoster) confirmedSeeds() []yacymodel.Seed {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]yacymodel.Seed(nil), s.reachableConfirmations...)
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

type stubAnnouncementObserver struct {
	rounds chan map[peerannouncement.ContactOutcome]int
}

func newStubAnnouncementObserver() *stubAnnouncementObserver {
	return &stubAnnouncementObserver{
		rounds: make(chan map[peerannouncement.ContactOutcome]int, 16),
	}
}

func (s *stubAnnouncementObserver) ObserveAnnounceRound(
	amountOfPeersPerContactOutcome map[peerannouncement.ContactOutcome]int,
) {
	s.rounds <- amountOfPeersPerContactOutcome
}

func announcerFor(
	self yacymodel.Seed,
	seeds []yacymodel.Seed,
	roster *stubRoster,
	reachableCap int,
	observer peerannouncement.AnnouncementObserver,
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
		observer,
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

func roundObservedFor(
	t *testing.T,
	self yacymodel.Seed,
	roster *stubRoster,
) map[peerannouncement.ContactOutcome]int {
	t.Helper()

	observer := newStubAnnouncementObserver()
	announcer := announcerFor(self, nil, roster, 4, observer)

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		announcer.Run(ctx)
	}()

	var round map[peerannouncement.ContactOutcome]int
	select {
	case round = <-observer.rounds:
	case <-time.After(confirmationWait):
		cancel()
		t.Fatal("timed out waiting for the announcer to observe a round")
	}

	cancel()
	<-stopped

	return round
}

func assertSolePeerOutcome(
	t *testing.T,
	round map[peerannouncement.ContactOutcome]int,
	want peerannouncement.ContactOutcome,
) {
	t.Helper()

	for _, outcome := range peerannouncement.ContactOutcomes() {
		amountOfPeers, reported := round[outcome]
		if !reported {
			t.Fatalf("outcome %q absent from the round, want every outcome reported", outcome)
		}
		wantAmountOfPeers := 0
		if outcome == want {
			wantAmountOfPeers = 1
		}
		if amountOfPeers != wantAmountOfPeers {
			t.Fatalf(
				"outcome %q = %d, want %d",
				outcome,
				amountOfPeers,
				wantAmountOfPeers,
			)
		}
	}
}

func TestAnnounceReportsPeerThatReachedThisNodeBack(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	peer := newStubPeer(t, "peer", seniorAnswer(t))

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})

	assertSolePeerOutcome(
		t,
		roundObservedFor(t, self.seed, roster),
		peerannouncement.ContactOutcomeFromReportedSelfType(yacymodel.PeerSenior),
	)
}

func TestAnnounceReportsPeerThatCouldNotReachThisNodeBack(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	peer := newStubPeer(t, "peer", juniorAnswer(t))

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})

	assertSolePeerOutcome(
		t,
		roundObservedFor(t, self.seed, roster),
		peerannouncement.ContactOutcomeFromReportedSelfType(yacymodel.PeerJunior),
	)
}

func TestAnnounceReportsPeerThatDoesNotKnowThisNode(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	peer := newStubPeer(t, "peer", virginAnswer(t))

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})

	assertSolePeerOutcome(
		t,
		roundObservedFor(t, self.seed, roster),
		peerannouncement.ContactOutcomeFromReportedSelfType(yacymodel.PeerVirgin),
	)
}

func TestAnnounceReportsPeerThatReportedNoTypeForThisNode(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	peer := newStubPeer(t, "peer", untypedAnswer(t))

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})

	assertSolePeerOutcome(
		t,
		roundObservedFor(t, self.seed, roster),
		peerannouncement.PeerReportedNoSelfType,
	)
}

func TestAnnounceReportsPeerThatDidNotAnswer(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	peer := newStubPeer(t, "peer", unavailableAnswer())

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})

	assertSolePeerOutcome(
		t,
		roundObservedFor(t, self.seed, roster),
		peerannouncement.PeerDidNotAnswer,
	)
}

func TestAnnounceRecordsReachableAndGossip(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	known := newStubPeer(t, "known", seniorAnswer(t))
	peer := newStubPeer(t, "peer", seniorAnswer(t, known.seed))

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})
	runUntilPeerConfirmed(
		t,
		announcerFor(self.seed, nil, roster, 4, peerannouncement.DiscardObserver),
		roster,
	)

	reachable := roster.reachableHashes()
	if len(reachable) != 1 || reachable[0] != peer.seed.Hash {
		t.Fatalf("reachable = %v, want [%v]", reachable, peer.seed.Hash)
	}
	confirmedSeeds := roster.confirmedSeeds()
	if len(confirmedSeeds) != 1 || confirmedSeeds[0].Hash != peer.seed.Hash {
		t.Fatalf("confirmed seeds = %v, want responding peer seed", confirmedSeeds)
	}

	discovered := roster.discoveredSeeds()
	if len(discovered) != 1 || discovered[0].Hash != known.seed.Hash {
		t.Fatalf("discovered = %v, want gossiped known seed", discovered)
	}
}

func TestAnnounceReplacesPeerIdentityAtKnownAddress(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	peer := newStubPeer(t, "peer", answerFromReplacementPeer(t))
	respondingPeer := answeringPeerSeed(t)

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})
	runUntilPeerConfirmed(
		t,
		announcerFor(self.seed, nil, roster, 4, peerannouncement.DiscardObserver),
		roster,
	)

	unreachable := roster.unreachableHashes()
	if len(unreachable) != 1 || unreachable[0] != peer.seed.Hash {
		t.Fatalf("unreachable = %v, want [%v]", unreachable, peer.seed.Hash)
	}
	reachable := roster.reachableHashes()
	if len(reachable) != 1 || reachable[0] != respondingPeer.Hash {
		t.Fatalf("reachable = %v, want new peer %v", reachable, respondingPeer.Hash)
	}
	confirmedSeeds := roster.confirmedSeeds()
	if len(confirmedSeeds) != 1 || !reflect.DeepEqual(confirmedSeeds[0], respondingPeer) {
		t.Fatalf("confirmed seeds = %v, want responding peer seed", confirmedSeeds)
	}
}

func TestAnnounceSkipsSelfInTargets(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	peer := newStubPeer(t, "peer", seniorAnswer(t))

	roster := newStubRoster(nil, []yacymodel.Seed{self.seed, peer.seed})
	runUntilPeerConfirmed(
		t,
		announcerFor(self.seed, nil, roster, 4, peerannouncement.DiscardObserver),
		roster,
	)

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
	runUntilPeerConfirmed(
		t,
		announcerFor(self.seed, nil, roster, 4, peerannouncement.DiscardObserver),
		roster,
	)

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
	peer := newStubPeer(t, "peer", untypedAnswer(t, known.seed))

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})
	runUntilPeerConfirmed(
		t,
		announcerFor(self.seed, nil, roster, 4, peerannouncement.DiscardObserver),
		roster,
	)

	unreachable := roster.unreachableHashes()
	if len(unreachable) != 1 || unreachable[0] != peer.seed.Hash {
		t.Fatalf("unreachable = %v, want [%v]", unreachable, peer.seed.Hash)
	}
	if discovered := roster.discoveredSeeds(); len(discovered) != 0 {
		t.Fatalf("discovered = %v, want no seeds from a peer outside our network", discovered)
	}
}

func TestAnnounceRejectsPeerThatAnswersVirgin(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	known := newStubPeer(t, "known", seniorAnswer(t))
	peer := newStubPeer(t, "peer", virginAnswer(t, known.seed))

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})
	runUntilPeerConfirmed(
		t,
		announcerFor(self.seed, nil, roster, 4, peerannouncement.DiscardObserver),
		roster,
	)

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
	runUntilPeerConfirmed(
		t,
		announcerFor(self.seed, nil, roster, 1, peerannouncement.DiscardObserver),
		roster,
	)

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
		announcerFor(
			self.seed,
			[]yacymodel.Seed{peer.seed},
			roster,
			4,
			peerannouncement.DiscardObserver,
		),
		roster,
	)

	discovered := roster.discoveredSeeds()
	if len(discovered) != 1 || discovered[0].Hash != peer.seed.Hash {
		t.Fatalf("discovered = %v, want the seed source seed on start", discovered)
	}
}
