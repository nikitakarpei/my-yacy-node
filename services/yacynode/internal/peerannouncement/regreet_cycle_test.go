package peerannouncement_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerannouncement"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const confirmationWait = 10 * time.Second

var errNothingListens = errors.New("nothing listens")

type stubRoster struct {
	mu                     sync.Mutex
	reachablePeers         []yacymodel.Seed
	unreachablePeerHashes  []yacymodel.Hash
	addresses              map[yacymodel.Hash]yacymodel.PeerAddresses
	discovered             []yacymodel.Seed
	reachable              []yacymodel.Hash
	unreachable            []yacymodel.Hash
	reachableConfirmations []yacymodel.Seed
	answeredBaseURLs       []peerwire.BaseURL
	confirmations          chan struct{}
}

func newStubRoster(reachable, unreachable []yacymodel.Seed) *stubRoster {
	addresses := make(map[yacymodel.Hash]yacymodel.PeerAddresses)
	for _, seed := range append(append([]yacymodel.Seed{}, reachable...), unreachable...) {
		if seedAddresses, addressable := seed.Addresses(); addressable {
			addresses[seed.Hash] = seedAddresses
		}
	}
	unreachablePeerHashes := make([]yacymodel.Hash, len(unreachable))
	for index, seed := range unreachable {
		unreachablePeerHashes[index] = seed.Hash
	}

	return &stubRoster{
		reachablePeers:        reachable,
		unreachablePeerHashes: unreachablePeerHashes,
		addresses:             addresses,
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

func (s *stubRoster) AddressesOf(
	_ context.Context,
	peer yacymodel.Hash,
) (yacymodel.PeerAddresses, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	addresses, found := s.addresses[peer]

	return addresses, found
}

func (s *stubRoster) Discover(_ context.Context, seeds ...yacymodel.Seed) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.discovered = append(s.discovered, seeds...)
}

func (s *stubRoster) ConfirmReachable(
	_ context.Context,
	seed yacymodel.Seed,
	answeredBaseURL peerwire.BaseURL,
) {
	s.mu.Lock()
	s.reachable = append(s.reachable, seed.Hash)
	s.reachableConfirmations = append(s.reachableConfirmations, seed)
	s.answeredBaseURLs = append(s.answeredBaseURLs, answeredBaseURL)
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

func (s *stubRoster) confirmedBaseURLs() []peerwire.BaseURL {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]peerwire.BaseURL(nil), s.answeredBaseURLs...)
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

type stubSeedSource struct {
	seeds []yacymodel.Seed
}

func (s *stubSeedSource) Fetch(context.Context) []yacymodel.Seed {
	return s.seeds
}

type observedContacts struct {
	amountOfPeersPerContactOutcome  map[peerannouncement.ContactOutcome]int
	amountOfReachablePeersPerScheme map[peerwire.Scheme]int
}

type stubAnnouncementObserver struct {
	observations chan observedContacts
}

func newStubAnnouncementObserver() *stubAnnouncementObserver {
	return &stubAnnouncementObserver{observations: make(chan observedContacts, 16)}
}

func (s *stubAnnouncementObserver) ObservePeerContacts(
	amountOfPeersPerContactOutcome map[peerannouncement.ContactOutcome]int,
	amountOfReachablePeersPerScheme map[peerwire.Scheme]int,
) {
	s.observations <- observedContacts{amountOfPeersPerContactOutcome, amountOfReachablePeersPerScheme}
}

type stoppedClock struct{}

func (stoppedClock) Now() time.Time { return time.Unix(1_700_000_000, 0).UTC() }

func (stoppedClock) After(time.Duration, func()) (stop func()) { return func() {} }

func exchangeOver(client *http.Client) peerwire.FormExchange {
	return peerwire.NewFormExchange(client, stoppedClock{})
}

type stubLearningSelf struct {
	mu                     sync.Mutex
	seed                   yacymodel.Seed
	adoptsReportedHost     bool
	reportsPerAnnouncement [][]yacymodel.Host
}

func (s *stubLearningSelf) SelfSeed(context.Context) yacymodel.Seed {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.seed
}

func (s *stubLearningSelf) ObserveReportedHosts(_ context.Context, reportedHosts []yacymodel.Host) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.reportsPerAnnouncement = append(s.reportsPerAnnouncement, reportedHosts)
	if s.adoptsReportedHost && len(reportedHosts) > 0 {
		s.seed.PrimaryAddress = yacymodel.Some(reportedHosts[0])
	}
}

func (s *stubLearningSelf) reportedHosts() [][]yacymodel.Host {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([][]yacymodel.Host(nil), s.reportsPerAnnouncement...)
}

func announcerFor(
	self yacymodel.Seed,
	seeds []yacymodel.Seed,
	roster *stubRoster,
	reachableCap int,
	observer peerannouncement.AnnouncementObserver,
) peerannouncement.Announcer {
	learningSelf := &stubLearningSelf{seed: self}

	return peerannouncement.New(
		configFor(reachableCap),
		exchangeOver(http.DefaultClient),
		learningSelf,
		learningSelf,
		&stubSeedSource{seeds: seeds},
		roster,
		observer,
	)
}

func configFor(reachableCap int) peerannouncement.Config {
	return peerannouncement.Config{
		NetworkName:        networkName,
		Interval:           time.Hour,
		ReachableCap:       reachableCap,
		ContactConcurrency: 4,
	}
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

func outcomesObservedFor(
	t *testing.T,
	self yacymodel.Seed,
	roster *stubRoster,
) map[peerannouncement.ContactOutcome]int {
	t.Helper()

	return lastContactsObservedWith(t, &stubLearningSelf{seed: self}, roster, 1)
}

func assertSolePeerOutcome(
	t *testing.T,
	amountOfPeersPerOutcome map[peerannouncement.ContactOutcome]int,
	want peerannouncement.ContactOutcome,
) {
	t.Helper()

	for _, outcome := range peerannouncement.ContactOutcomes() {
		amountOfPeers, reported := amountOfPeersPerOutcome[outcome]
		if !reported {
			t.Fatalf(
				"outcome %q absent from the observed contacts, want every outcome reported",
				outcome,
			)
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
		outcomesObservedFor(t, self.seed, roster),
		peerannouncement.ContactOutcomeFromReportedSelfType(yacymodel.PeerSenior),
	)
}

func TestAnnounceReportsPeerThatCouldNotReachThisNodeBack(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	peer := newStubPeer(t, "peer", juniorAnswer(t))

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})

	assertSolePeerOutcome(
		t,
		outcomesObservedFor(t, self.seed, roster),
		peerannouncement.ContactOutcomeFromReportedSelfType(yacymodel.PeerJunior),
	)
}

func TestAnnounceReportsPeerThatDoesNotKnowThisNode(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	peer := newStubPeer(t, "peer", virginAnswer(t))

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})

	assertSolePeerOutcome(
		t,
		outcomesObservedFor(t, self.seed, roster),
		peerannouncement.ContactOutcomeFromReportedSelfType(yacymodel.PeerVirgin),
	)
}

func TestAnnounceReportsPeerThatReportedNoTypeForThisNode(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	peer := newStubPeer(t, "peer", untypedAnswer(t))

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})

	assertSolePeerOutcome(
		t,
		outcomesObservedFor(t, self.seed, roster),
		peerannouncement.PeerReportedNoSelfType,
	)
}

func TestAnnounceReportsPeerThatDidNotAnswer(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	peer := newStubPeer(t, "peer", unavailableAnswer())

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})

	assertSolePeerOutcome(
		t,
		outcomesObservedFor(t, self.seed, roster),
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

func TestAnnounceTreatsAReplyFromAnotherPeerAsNoAnswer(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	peer := newStubPeer(t, "peer", answerFromReplacementPeer(t))

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
		t.Fatalf("reachable = %v, want no peer confirmed by another peer's reply", reachable)
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

func lastContactsObservedWith(
	t *testing.T,
	self *stubLearningSelf,
	roster *stubRoster,
	amountOfAnnouncements int,
) map[peerannouncement.ContactOutcome]int {
	t.Helper()

	amountOfPeersPerOutcome, _ := lastContactsObservedOver(
		t,
		http.DefaultClient,
		self,
		roster,
		amountOfAnnouncements,
	)

	return amountOfPeersPerOutcome
}

func lastContactsObservedOver(
	t *testing.T,
	client *http.Client,
	self *stubLearningSelf,
	roster *stubRoster,
	amountOfAnnouncements int,
) (map[peerannouncement.ContactOutcome]int, map[peerwire.Scheme]int) {
	t.Helper()

	observer := newStubAnnouncementObserver()
	announcer := peerannouncement.New(
		configFor(4),
		exchangeOver(client),
		self,
		self,
		&stubSeedSource{},
		roster,
		observer,
	)

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		announcer.Run(ctx)
	}()

	var observed observedContacts
	for range amountOfAnnouncements {
		select {
		case observed = <-observer.observations:
		case <-time.After(confirmationWait):
			cancel()
			t.Fatal("timed out waiting for the announcer to observe its peer contacts")
		}
	}

	cancel()
	<-stopped

	return observed.amountOfPeersPerContactOutcome, observed.amountOfReachablePeersPerScheme
}

func TestAnnounceReportsTheHostsThatPeersSaw(t *testing.T) {
	self := &stubLearningSelf{seed: newStubPeer(t, "self", seniorAnswer(t)).seed}
	first := newStubPeer(t, "first", seniorAnswer(t))
	second := newStubPeer(t, "second", juniorAnswer(t))

	lastContactsObservedWith(
		t,
		self,
		newStubRoster(nil, []yacymodel.Seed{first.seed, second.seed}),
		1,
	)

	reported := self.reportedHosts()
	if len(reported) != 1 || len(reported[0]) != 2 {
		t.Fatalf("reported hosts = %v, want one report of two hosts", reported)
	}
	for _, host := range reported[0] {
		if host.String() != "203.0.113.9" {
			t.Fatalf("reported host = %v, want the host the peers saw", host)
		}
	}
}

func TestAnnounceReportsNoHostFromAPeerThatDidNotAnswer(t *testing.T) {
	self := &stubLearningSelf{seed: newStubPeer(t, "self", seniorAnswer(t)).seed}
	peer := newStubPeer(t, "peer", unavailableAnswer())

	lastContactsObservedWith(t, self, newStubRoster(nil, []yacymodel.Seed{peer.seed}), 1)

	if reported := self.reportedHosts(); len(reported) != 1 || len(reported[0]) != 0 {
		t.Fatalf("reported hosts = %v, want one report without hosts", reported)
	}
}

func TestAnnounceReportsAMappedIPv4HostAsTheIPv4Host(t *testing.T) {
	self := &stubLearningSelf{seed: newStubPeer(t, "self", seniorAnswer(t)).seed}
	peer := newStubPeer(t, "peer", answerReporting("::ffff:203.0.113.9"))

	lastContactsObservedWith(t, self, newStubRoster(nil, []yacymodel.Seed{peer.seed}), 1)

	reported := self.reportedHosts()
	if len(reported) != 1 || len(reported[0]) != 1 || reported[0][0].String() != "203.0.113.9" {
		t.Fatalf("reported hosts = %v, want [[203.0.113.9]]", reported)
	}
}

func TestAnnounceAgainWhenTheSeedAddressChanges(t *testing.T) {
	self := &stubLearningSelf{
		seed:               newStubPeer(t, "self", seniorAnswer(t)).seed,
		adoptsReportedHost: true,
	}
	peer := newStubPeer(t, "peer", seniorAnswer(t))

	lastContactsObservedWith(t, self, newStubRoster(nil, []yacymodel.Seed{peer.seed}), 2)

	if peer.greetCount() != 2 {
		t.Fatalf("peer greeted %d times, want 2 within one interval", peer.greetCount())
	}
}

func TestAnnounceCountsConfirmedPeersPerScheme(t *testing.T) {
	self := &stubLearningSelf{seed: newStubPeer(t, "self", seniorAnswer(t)).seed}
	reachingPeer := newStubPeer(t, "reaching", seniorAnswer(t))
	failingPeer := newStubPeer(t, "failing", unavailableAnswer())

	_, schemes := lastContactsObservedOver(
		t,
		http.DefaultClient,
		self,
		newStubRoster(nil, []yacymodel.Seed{reachingPeer.seed, failingPeer.seed}),
		1,
	)

	want := map[peerwire.Scheme]int{peerwire.HTTP: 1, peerwire.HTTPS: 0}
	if !reflect.DeepEqual(schemes, want) {
		t.Fatalf("confirmed peers per scheme = %v, want %v", schemes, want)
	}
}

func TestAnnounceConfirmsAPeerWithTheBaseURLThatAnswered(t *testing.T) {
	self := &stubLearningSelf{seed: newStubPeer(t, "self", seniorAnswer(t)).seed}
	peer := newStubPeer(t, "reaching", seniorAnswer(t))
	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})

	lastContactsObservedOver(t, http.DefaultClient, self, roster, 1)

	host, _ := peer.seed.PrimaryAddress.Get()
	port, _ := peer.seed.Port.Get()
	want := peerwire.BaseURL{Scheme: peerwire.HTTP, Host: host, Port: port}
	if got := roster.confirmedBaseURLs(); len(got) != 1 || got[0] != want {
		t.Fatalf("confirmed base URLs = %v, want %v", got, want)
	}
}

func TestAnnounceTakesNoReportedHostFromAPeerReachedByName(t *testing.T) {
	self := &stubLearningSelf{seed: newStubPeer(t, "self", seniorAnswer(t)).seed}
	peerSeed := peerSeedAt(t, "secure", "example.com", 8090)
	server := httptest.NewTLSServer(seniorAnswer(t)(peerSeed))
	t.Cleanup(server.Close)
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	client := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, protocol, address string) (net.Conn, error) {
			if address != "example.com:443" {
				return nil, errNothingListens
			}
			var dialer net.Dialer

			conn, err := dialer.DialContext(ctx, protocol, server.Listener.Addr().String())
			if err != nil {
				return nil, fmt.Errorf("dial: %w", err)
			}

			return conn, nil
		},
		TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
	}}

	_, schemes := lastContactsObservedOver(
		t,
		client,
		self,
		newStubRoster(nil, []yacymodel.Seed{peerSeed}),
		1,
	)

	if schemes[peerwire.HTTPS] != 1 {
		t.Fatalf("confirmed peers per scheme = %v, want one over https", schemes)
	}
	if reported := self.reportedHosts(); len(reported) != 1 || len(reported[0]) != 0 {
		t.Fatalf("reported hosts = %v, want none from a peer reached by name", reported)
	}
}

func TestAnnounceTreatsAReplyWithAnUnaddressableSeedAsNoAnswer(t *testing.T) {
	self := newStubPeer(t, "self", seniorAnswer(t))
	peer := newStubPeer(t, "peer", func(respondingSeed yacymodel.Seed) http.HandlerFunc {
		respondingSeed.PrimaryAddress = yacymodel.None[yacymodel.Host]()

		return helloAnswer(yacyproto.HelloResponse{
			YourIP:   "203.0.113.9",
			YourType: yacymodel.Some(yacymodel.PeerSenior),
			Seeds:    []yacymodel.Seed{respondingSeed},
		})
	})

	roster := newStubRoster(nil, []yacymodel.Seed{peer.seed})
	runUntilPeerConfirmed(
		t,
		announcerFor(self.seed, nil, roster, 4, peerannouncement.DiscardObserver),
		roster,
	)

	if unreachable := roster.unreachableHashes(); len(unreachable) != 1 ||
		unreachable[0] != peer.seed.Hash {
		t.Fatalf("unreachable = %v, want [%v]", unreachable, peer.seed.Hash)
	}
}
