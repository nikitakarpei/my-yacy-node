package postingoffer_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingoffer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/replicaeligibility"
)

const (
	postingOffersRedundancy = 1
	recipientCooldown       = time.Minute
)

type postingOfferOptions struct {
	postings     map[yacymodel.Hash]yacymodel.RWIPosting
	reachability postingoffer.Reachability
	observer     postingoffer.Observer
	self         yacymodel.Hash
	redundancy   int
}

type postingOfferHarness struct {
	vault         *vault.Vault
	schedule      *postingofferschedule.Schedule
	replicas      *postingreplicas.Replicas
	eligibility   *replicaeligibility.Peers
	postingOffers *postingoffer.PostingOffers
}

func openPostingOffers(t *testing.T, options postingOfferOptions) postingOfferHarness {
	t.Helper()

	options = withPostingOfferDefaults(options)

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("memoryvault.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	schedule, err := postingofferschedule.Open(v, frozenNow, discardedScheduleObservations{})
	if err != nil {
		t.Fatalf("postingofferschedule.Open: %v", err)
	}
	replicas, err := postingreplicas.Open(v, schedule)
	if err != nil {
		t.Fatalf("postingreplicas.Open: %v", err)
	}
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}
	eligibility := replicaeligibility.New(recipientCooldown, frozenNow)

	return postingOfferHarness{
		vault:       v,
		schedule:    schedule,
		replicas:    replicas,
		eligibility: eligibility,
		postingOffers: postingoffer.New(
			v,
			schedule,
			replicas,
			fakePostingIndex{postings: options.postings},
			options.reachability,
			eligibility,
			options.observer,
			partitions,
			options.self,
			options.redundancy,
		),
	}
}

func withPostingOfferDefaults(options postingOfferOptions) postingOfferOptions {
	if options.reachability == nil {
		options.reachability = fakeReachability{}
	}
	if options.observer == nil {
		options.observer = discardedRingSectorObservations{}
	}
	if options.self.IsZero() {
		options.self = thisNodeFartherThanEveryPeer()
	}
	if options.redundancy == 0 {
		options.redundancy = postingOffersRedundancy
	}

	return options
}

func (h postingOfferHarness) holdBack(peer yacymodel.Hash) {
	h.eligibility.OfferDeclined(peer, 0)
}

func (h postingOfferHarness) storePosting(
	t *testing.T,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return h.schedule.PostingStored(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingStored: %v", err)
	}
}

func (h postingOfferHarness) recordAccepted(
	t *testing.T,
	peer yacymodel.Hash,
	postings ...yacymodel.RWIPosting,
) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return h.replicas.RecordAccepted(tx, peer, postings)
	}); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}
}

func (h postingOfferHarness) holdersOf(
	t *testing.T,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) []yacymodel.Hash {
	t.Helper()

	var holders []yacymodel.Hash
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		holders, err = h.replicas.HoldersOf(tx, postingidentity.IdentityOf(word, url))

		return err
	}); err != nil {
		t.Fatalf("HoldersOf: %v", err)
	}

	return holders
}

func (h postingOfferHarness) dueNow(
	t *testing.T,
	peers []yacymodel.Seed,
) ([]postingoffer.PostingOffer, []postingidentity.Identity) {
	t.Helper()

	offers, gonePostings, err := h.postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("DueNow: %v", err)
	}

	return offers, gonePostings
}

func (h postingOfferHarness) dueOffer(
	t *testing.T,
	peers []yacymodel.Seed,
) postingoffer.PostingOffer {
	t.Helper()

	offers, _ := h.dueNow(t, peers)
	if len(offers) != 1 {
		t.Fatalf("offers = %+v, want a single posting", offers)
	}

	return offers[0]
}

func frozenNow() time.Time { return time.Unix(1000, 0) }

func thisNodeFartherThanEveryPeer() yacymodel.Hash { return yacymodel.WordHash("self5") }

func thisNodeCloserThanEveryPeer() yacymodel.Hash { return yacymodel.WordHash("self22") }

type fakePostingIndex struct {
	postings map[yacymodel.Hash]yacymodel.RWIPosting
}

func (f fakePostingIndex) RWICount(*vault.Txn) (int, error) { return len(f.postings), nil }

func (f fakePostingIndex) PostingOf(
	_ context.Context,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	entry, found := f.postings[fakePostingKey(word, url)]

	return entry, found, nil
}

func (f fakePostingIndex) ScanWord(
	context.Context,
	yacymodel.Hash,
	func(yacymodel.RWIPosting) (bool, error),
) error {
	return nil
}

func fakePostingKey(word yacymodel.Hash, url yacymodel.URLHash) yacymodel.Hash {
	return yacymodel.WordHash(word.String() + url.String())
}

func urlHash() yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash("u1").String())
	if err != nil {
		panic(err)
	}

	return hash
}

func fakePosting(word yacymodel.Hash, url yacymodel.URLHash) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{WordHash: word, URLHash: url}
}

func storedPostings(postings ...yacymodel.RWIPosting) map[yacymodel.Hash]yacymodel.RWIPosting {
	byKey := make(map[yacymodel.Hash]yacymodel.RWIPosting, len(postings))
	for _, posting := range postings {
		byKey[fakePostingKey(posting.WordHash, posting.URLHash)] = posting
	}

	return byKey
}

type fakeReachability struct {
	reachable         []yacymodel.Seed
	recentlyReachable map[yacymodel.Hash]struct{}
}

func (f fakeReachability) IsReachable(_ context.Context, peer yacymodel.Hash) bool {
	for _, seed := range f.reachable {
		if seed.Hash == peer {
			return true
		}
	}

	return false
}

func (f fakeReachability) IsRecentlyReachable(_ context.Context, peer yacymodel.Hash) bool {
	_, recent := f.recentlyReachable[peer]

	return recent
}

func seed(hash yacymodel.Hash) yacymodel.Seed {
	host, err := yacymodel.ParseHost("192.0.2.1")
	if err != nil {
		panic(err)
	}
	port, err := yacymodel.ParsePort("8090")
	if err != nil {
		panic(err)
	}

	return yacymodel.Seed{
		Hash:           hash,
		PrimaryAddress: yacymodel.Some(host),
		Port:           yacymodel.Some(port),
		Capabilities:   yacymodel.Some(yacymodel.PeerCapabilities{AcceptRemoteIndex: true}),
	}
}

func peerSeeds(names ...string) []yacymodel.Seed {
	seeds := make([]yacymodel.Seed, 0, len(names))
	for _, name := range names {
		seeds = append(seeds, seed(yacymodel.WordHash(name)))
	}

	return seeds
}

func indexDecliningSeed(hash yacymodel.Hash) yacymodel.Seed {
	declining := seed(hash)
	declining.Capabilities = yacymodel.Some(yacymodel.PeerCapabilities{AcceptRemoteIndex: false})

	return declining
}

func otherThan(peers []yacymodel.Seed, closest yacymodel.Hash) yacymodel.Hash {
	for _, peer := range peers {
		if peer.Hash != closest {
			return peer.Hash
		}
	}

	panic("no peer other than the closest")
}

type discardedRingSectorObservations struct{}

func (discardedRingSectorObservations) ObservePeersAcceptingRemoteIndexPerDHTRingSector([]int) {}

type recordedRingSectorObservations struct {
	peersPerSector [][]int
}

func (r *recordedRingSectorObservations) ObservePeersAcceptingRemoteIndexPerDHTRingSector(
	peersPerSector []int,
) {
	r.peersPerSector = append(r.peersPerSector, peersPerSector)
}

type discardedScheduleObservations struct{}

func (discardedScheduleObservations) ObserveScheduledPostings(int) {}

func (discardedScheduleObservations) ObserveLongestOfferLateness(time.Duration) {}

func TestDueReportsMissingReplicaForDuePosting(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	peers := peerSeeds("peer")
	harness := openPostingOffers(t, postingOfferOptions{
		postings: storedPostings(fakePosting(word, url)),
	})

	harness.storePosting(t, word, url)

	offers, gonePostings := harness.dueNow(t, peers)
	if len(offers) != 1 || len(gonePostings) != 0 {
		t.Fatalf("offers = %+v, want a single posting and no gone entries", offers)
	}
	missing := offers[0]
	if missing.AcceptancesNeeded != 1 || len(missing.Peers) != 1 ||
		missing.Peers[0].Hash != peers[0].Hash {
		t.Fatalf("missing = %+v, want one replica needed from %v", missing, peers[0].Hash)
	}
}

func TestDueRenewsTheHeldReplicaWhenRedundancyMet(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	peers := peerSeeds("peer")
	harness := openPostingOffers(t, postingOfferOptions{
		postings:     storedPostings(fakePosting(word, url)),
		reachability: fakeReachability{reachable: peers},
	})

	harness.storePosting(t, word, url)
	harness.recordAccepted(t, peers[0].Hash, fakePosting(word, url))

	offer := harness.dueOffer(t, peers)
	if offer.AcceptancesNeeded != 0 {
		t.Fatalf("AcceptancesNeeded = %d, want zero", offer.AcceptancesNeeded)
	}
	if len(offer.Peers) != 1 || offer.Peers[0].Hash != peers[0].Hash {
		t.Fatalf("Peers = %v, want the peer that holds the replica", offer.Peers)
	}
}

func TestDueOwesNoReplicasWhenNoPeerIsCloserThanThisNode(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	harness := openPostingOffers(t, postingOfferOptions{
		postings: storedPostings(fakePosting(word, url)),
	})

	harness.storePosting(t, word, url)

	missing := harness.dueOffer(t, nil)
	if missing.AcceptancesNeeded != 0 || len(missing.Peers) != 0 {
		t.Fatalf(
			"missing = %+v, want no replicas owed: this node holds the only one the DHT asks for",
			missing,
		)
	}
}

func TestDueLowersReplicasOwedWhenThisNodeIsResponsible(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	peers := peerSeeds("p1", "p2", "p3")
	harness := openPostingOffers(t, postingOfferOptions{
		postings:     storedPostings(fakePosting(word, url)),
		reachability: fakeReachability{reachable: peers},
		self:         thisNodeCloserThanEveryPeer(),
		redundancy:   3,
	})

	harness.storePosting(t, word, url)

	missing := harness.dueOffer(t, peers)
	if missing.AcceptancesNeeded != 2 || len(missing.Peers) != 2 {
		t.Fatalf(
			"missing = %+v, want two replicas owed: this node is one of the three the DHT"+
				" makes responsible",
			missing,
		)
	}
}

func TestDueLeavesReplicasOwedWhenThisNodeIsOutsideTheWindow(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	peers := peerSeeds("p1", "p2", "p3")
	harness := openPostingOffers(t, postingOfferOptions{
		postings:     storedPostings(fakePosting(word, url)),
		reachability: fakeReachability{reachable: peers},
		redundancy:   3,
	})

	harness.storePosting(t, word, url)

	missing := harness.dueOffer(t, peers)
	if missing.AcceptancesNeeded != 3 || len(missing.Peers) != 3 {
		t.Fatalf(
			"missing = %+v, want three replicas owed: this node is farther than every peer",
			missing,
		)
	}
}

func TestDueNarrowsTheLedgerWindowWhenThisNodeIsResponsible(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	peers := peerSeeds("p1", "p2")
	harness := openPostingOffers(t, postingOfferOptions{
		postings:     storedPostings(fakePosting(word, url)),
		reachability: fakeReachability{reachable: peers},
		self:         thisNodeCloserThanEveryPeer(),
		redundancy:   2,
	})

	harness.storePosting(t, word, url)
	for _, peer := range peers {
		harness.recordAccepted(t, peer.Hash, fakePosting(word, url))
	}

	offer := harness.dueOffer(t, peers)
	if len(offer.StaleHolders) != 1 {
		t.Fatalf(
			"StaleHolders = %+v, want the one holder beyond the single replica peers owe",
			offer.StaleHolders,
		)
	}
}

func TestDueCollectsGoneForRemovedPosting(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	harness := openPostingOffers(t, postingOfferOptions{})

	harness.storePosting(t, word, url)

	offers, gonePostings := harness.dueNow(t, peerSeeds("peer"))
	if len(offers) != 0 || len(gonePostings) != 1 || gonePostings[0].Word != word {
		t.Fatalf("offers = %+v, want a single gone entry for %v", offers, word)
	}
}

func TestDueReportsStaleReplicaWithoutDroppingIt(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	stalePeer := yacymodel.WordHash("stale")
	peers := peerSeeds("fresh")
	harness := openPostingOffers(t, postingOfferOptions{
		postings: storedPostings(fakePosting(word, url)),
	})

	harness.storePosting(t, word, url)
	harness.recordAccepted(t, stalePeer, fakePosting(word, url))

	missing := harness.dueOffer(t, peers)
	if len(missing.Peers) != 1 || missing.Peers[0].Hash != peers[0].Hash {
		t.Fatalf(
			"missing = %+v, want an offer to %v regardless of stale peer %v",
			missing, peers[0].Hash, stalePeer,
		)
	}
	if len(missing.StaleHolders) != 1 || missing.StaleHolders[0] != stalePeer {
		t.Fatalf("StaleHolders = %+v, want the one stale peer %v", missing.StaleHolders, stalePeer)
	}
	stored := harness.holdersOf(t, word, url)
	if len(stored) != 1 || stored[0] != stalePeer {
		t.Fatalf("replicas = %v, want [%v] unchanged by DueNow", stored, stalePeer)
	}
}

func TestDueOffersOnlyAsManyReplicasAsAreNeeded(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	absentHolder := yacymodel.WordHash("absent")
	harness := openPostingOffers(t, postingOfferOptions{
		postings: storedPostings(fakePosting(word, url)),
		reachability: fakeReachability{
			recentlyReachable: map[yacymodel.Hash]struct{}{absentHolder: {}},
		},
		redundancy: 3,
	})

	harness.storePosting(t, word, url)
	harness.recordAccepted(t, absentHolder, fakePosting(word, url))

	missing := harness.dueOffer(t, peerSeeds("p1", "p2", "p3"))
	if missing.AcceptancesNeeded != 2 {
		t.Fatalf(
			"AcceptancesNeeded = %d, want 2 with one credible holder of three",
			missing.AcceptancesNeeded,
		)
	}
	if len(missing.Peers) != missing.AcceptancesNeeded {
		t.Fatalf(
			"Peers = %d seeds, want %d so accepting every offer stays within redundancy",
			len(missing.Peers),
			missing.AcceptancesNeeded,
		)
	}
}

func TestDueKeepsHolderThatStoppedAcceptingIndex(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	peer := yacymodel.WordHash("peer")
	declining := []yacymodel.Seed{indexDecliningSeed(peer)}
	harness := openPostingOffers(t, postingOfferOptions{
		postings:     storedPostings(fakePosting(word, url)),
		reachability: fakeReachability{reachable: declining},
	})

	harness.storePosting(t, word, url)
	harness.recordAccepted(t, peer, fakePosting(word, url))

	missing := harness.dueOffer(t, declining)
	if len(missing.StaleHolders) != 0 {
		t.Fatalf(
			"StaleHolders = %+v, want none: a peer that stops accepting a remote index "+
				"still serves the replica it holds",
			missing.StaleHolders,
		)
	}
	if missing.AcceptancesNeeded != 0 || len(missing.Peers) != 0 {
		t.Fatalf(
			"missing = %+v, want no further replicas and no offer to the holder",
			missing,
		)
	}
}

func TestDueOffersToNextPeerWhenClosestIsHeldBack(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	peers := peerSeeds("p1", "p2")
	harness := openPostingOffers(t, postingOfferOptions{
		postings:     storedPostings(fakePosting(word, url)),
		reachability: fakeReachability{reachable: peers},
	})

	harness.storePosting(t, word, url)

	first := harness.dueOffer(t, peers)
	if len(first.Peers) != 1 {
		t.Fatalf("offer = %+v, want one posting offered to the closest peer", first)
	}
	closest := first.Peers[0].Hash

	harness.holdBack(closest)

	second := harness.dueOffer(t, peers)
	if len(second.Peers) != 1 {
		t.Fatalf("offer = %+v, want one posting still offered somewhere", second)
	}
	if next := second.Peers[0].Hash; next == closest {
		t.Fatalf(
			"Peers[0] = %v, want a peer other than the held-back closest peer %v",
			next, closest,
		)
	}
}

func TestDueKeepsHolderOnHeldBackPeer(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	peers := peerSeeds("peer")
	harness := openPostingOffers(t, postingOfferOptions{
		postings:     storedPostings(fakePosting(word, url)),
		reachability: fakeReachability{reachable: peers},
	})

	harness.storePosting(t, word, url)
	harness.recordAccepted(t, peers[0].Hash, fakePosting(word, url))
	harness.holdBack(peers[0].Hash)

	offer := harness.dueOffer(t, peers)
	if len(offer.StaleHolders) != 0 {
		t.Fatalf(
			"StaleHolders = %+v, want none: a peer in cooldown still serves the replica it holds",
			offer.StaleHolders,
		)
	}
	if offer.AcceptancesNeeded != 0 {
		t.Fatalf("AcceptancesNeeded = %d, want no further replicas", offer.AcceptancesNeeded)
	}
	if len(offer.Peers) != 1 || offer.Peers[0].Hash != peers[0].Hash {
		t.Fatalf("Peers = %v, want the held-back peer that holds the replica", offer.Peers)
	}
}

func TestDueNarrowsResponsibilityWindowToPeersInCooldown(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	peers := peerSeeds("p1", "p2", "p3")
	harness := openPostingOffers(t, postingOfferOptions{
		postings:     storedPostings(fakePosting(word, url)),
		reachability: fakeReachability{reachable: peers},
	})

	harness.storePosting(t, word, url)

	closest := harness.dueOffer(t, peers).Peers[0].Hash
	harness.holdBack(closest)

	eligible := harness.dueOffer(t, peers)
	harness.recordAccepted(t, eligible.Peers[0].Hash, fakePosting(word, url))

	offer := harness.dueOffer(t, peers)
	if offer.AcceptancesNeeded != 1 {
		t.Fatalf(
			"AcceptancesNeeded = %d, want one replica needed: a peer in cooldown does not widen"+
				" the responsibility window",
			offer.AcceptancesNeeded,
		)
	}
	if len(offer.StaleHolders) != 0 {
		t.Fatalf(
			"StaleHolders = %+v, want none while the posting is below redundancy",
			offer.StaleHolders,
		)
	}
}

func TestDueKeepsHolderOutsideWindowUntilRedundancyIsMet(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	peers := peerSeeds("p1", "p2")
	harness := openPostingOffers(t, postingOfferOptions{
		postings:     storedPostings(fakePosting(word, url)),
		reachability: fakeReachability{reachable: peers},
	})

	harness.storePosting(t, word, url)

	closest := harness.dueOffer(t, peers).Peers[0].Hash
	outside := otherThan(peers, closest)

	harness.recordAccepted(t, outside, fakePosting(word, url))

	kept := harness.dueOffer(t, peers)
	if len(kept.StaleHolders) != 0 {
		t.Fatalf(
			"StaleHolders = %+v, want none: a reachable holder outside the window still serves it",
			kept.StaleHolders,
		)
	}

	harness.recordAccepted(t, closest, fakePosting(word, url))

	dropped := harness.dueOffer(t, peers)
	if len(dropped.StaleHolders) != 1 || dropped.StaleHolders[0] != outside {
		t.Fatalf(
			"StaleHolders = %+v, want [%v] once redundancy is met",
			dropped.StaleHolders, outside,
		)
	}
}

func TestDueDropsUnreachableHolderBelowRedundancy(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	ghost := yacymodel.WordHash("ghost")
	peers := peerSeeds("p1")
	harness := openPostingOffers(t, postingOfferOptions{
		postings:     storedPostings(fakePosting(word, url)),
		reachability: fakeReachability{reachable: peers},
	})

	harness.storePosting(t, word, url)
	harness.recordAccepted(t, ghost, fakePosting(word, url))

	offer := harness.dueOffer(t, peers)
	if len(offer.StaleHolders) != 1 || offer.StaleHolders[0] != ghost {
		t.Fatalf("StaleHolders = %+v, want [%v]: the peer is gone", offer.StaleHolders, ghost)
	}
	if offer.AcceptancesNeeded != 1 {
		t.Fatalf("AcceptancesNeeded = %d, want one replica needed", offer.AcceptancesNeeded)
	}
}

func TestDueReportsAcceptingPeersPerRingSector(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	observer := &recordedRingSectorObservations{}
	harness := openPostingOffers(t, postingOfferOptions{
		postings: storedPostings(fakePosting(word, url)),
		observer: observer,
	})
	peers := []yacymodel.Seed{
		seed(yacymodel.WordHash("peer")),
		indexDecliningSeed(yacymodel.WordHash("declining")),
	}

	harness.storePosting(t, word, url)
	harness.dueNow(t, peers)

	if len(observer.peersPerSector) != 1 {
		t.Fatalf("reports = %d, want one report per DueNow", len(observer.peersPerSector))
	}
	perSector := observer.peersPerSector[0]
	if len(perSector) != int(yacymodel.MaxDHTRingSector)+1 {
		t.Fatalf("sectors = %d, want %d", len(perSector), yacymodel.MaxDHTRingSector+1)
	}

	var acceptingPeers int
	for _, peers := range perSector {
		acceptingPeers += peers
	}
	if acceptingPeers != 1 {
		t.Errorf("accepting peers = %d, want only the peer accepting remote index", acceptingPeers)
	}
}

func TestDueCarriesThePostingPosition(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash()
	peers := peerSeeds("peer")
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}
	wantPosition := yacymodel.DHTRingPositionOfPosting(fakePosting(word, url), partitions)

	for name, reachability := range map[string]fakeReachability{
		"replica needed": {},
		"replica held":   {reachable: peers},
	} {
		t.Run(name, func(t *testing.T) {
			harness := openPostingOffers(t, postingOfferOptions{
				postings:     storedPostings(fakePosting(word, url)),
				reachability: reachability,
			})
			harness.storePosting(t, word, url)
			if len(reachability.reachable) > 0 {
				harness.recordAccepted(t, peers[0].Hash, fakePosting(word, url))
			}

			offer := harness.dueOffer(t, peers)
			if offer.PostingPosition != wantPosition {
				t.Errorf(
					"PostingPosition = %d, want %d", offer.PostingPosition, wantPosition,
				)
			}
		})
	}
}
