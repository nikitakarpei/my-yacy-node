package replicashortfall

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type fakePostingIndex struct {
	postings map[yacymodel.Hash]yacymodel.RWIPosting
	unread   error
}

func (f fakePostingIndex) RWICount(context.Context) (int, error) { return len(f.postings), nil }

func (f fakePostingIndex) Posting(
	_ context.Context,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	if f.unread != nil {
		return yacymodel.RWIPosting{}, false, f.unread
	}
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

type fakeReachability struct {
	reachable         []yacymodel.Seed
	recentlyReachable map[yacymodel.Hash]struct{}
}

func (f fakeReachability) Reachable(_ context.Context, peer yacymodel.Hash) bool {
	for _, seed := range f.reachable {
		if seed.Hash == peer {
			return true
		}
	}

	return false
}

func (f fakeReachability) RecentlyReachable(_ context.Context, peer yacymodel.Hash) bool {
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

func indexDecliningSeed(hash yacymodel.Hash) yacymodel.Seed {
	declining := seed(hash)
	declining.Capabilities = yacymodel.Some(yacymodel.PeerCapabilities{AcceptRemoteIndex: false})

	return declining
}

type everyPeerEligible struct{}

func (everyPeerEligible) Eligible(yacymodel.Hash) bool { return true }

type peersHeldBack map[yacymodel.Hash]struct{}

func (p peersHeldBack) Eligible(peer yacymodel.Hash) bool {
	_, held := p[peer]

	return !held
}

const shortfallRedundancy = 1

func thisNodeFartherThanEveryPeer() yacymodel.Hash { return yacymodel.WordHash("self5") }

func thisNodeCloserThanEveryPeer() yacymodel.Hash { return yacymodel.WordHash("self22") }

func recordAccepted(
	t *testing.T,
	v *vault.Vault,
	replicas *postingreplicas.Replicas,
	peer yacymodel.Hash,
	postings ...yacymodel.RWIPosting,
) {
	t.Helper()

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return replicas.RecordAccepted(tx, peer, postings)
	}); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}
}

func openShortfall(
	t *testing.T,
	now func() time.Time,
	postings map[yacymodel.Hash]yacymodel.RWIPosting,
	reachability Reachability,
) (*vault.Vault, *postingschedule.Schedule, *postingreplicas.Replicas, *Shortfall) {
	t.Helper()

	v, err := memvault.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	schedule, err := postingschedule.Open(v, now)
	if err != nil {
		t.Fatalf("postingschedule.Open: %v", err)
	}
	replicas, err := postingreplicas.Open(v, schedule)
	if err != nil {
		t.Fatalf("postingreplicas.Open: %v", err)
	}
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}

	shortfall := New(
		schedule,
		replicas,
		fakePostingIndex{postings: postings},
		reachability,
		everyPeerEligible{},
		partitions,
		thisNodeFartherThanEveryPeer(),
		shortfallRedundancy,
	)

	return v, schedule, replicas, shortfall
}

func store(
	t *testing.T,
	v *vault.Vault,
	schedule *postingschedule.Schedule,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) {
	t.Helper()

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return schedule.PostingStored(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingStored: %v", err)
	}
}

func TestDueReportsMissingReplicaForDuePosting(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	v, schedule, _, shortfall := openShortfall(
		t, func() time.Time { return now }, postings, fakeReachability{},
	)
	peers := []yacymodel.Seed{seed(peer)}

	store(t, v, schedule, word, url)

	due, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due.Offers) != 1 || len(due.Gone) != 0 {
		t.Fatalf("due = %+v, want a single posting and no gone entries", due)
	}
	missing := due.Offers[0]
	if missing.ReplicasNeeded != 1 || len(missing.Seeds) != 1 ||
		missing.Seeds[0].Hash != peer {
		t.Fatalf("missing = %+v, want one replica needed from %v", missing, peer)
	}
}

func TestDueRenewsTheHeldReplicaWhenRedundancyMet(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{seed(peer)}
	v, schedule, replicas, shortfall := openShortfall(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, peer, fakePosting(word, url))

	due, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due.Offers) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	offer := due.Offers[0]
	if offer.ReplicasNeeded != 0 {
		t.Fatalf("ReplicasNeeded = %d, want zero", offer.ReplicasNeeded)
	}
	if len(offer.Seeds) != 1 || offer.Seeds[0].Hash != peer {
		t.Fatalf("Seeds = %v, want the peer that holds the replica", offer.Seeds)
	}
}

func TestDueRenewsTheHeldReplicaOfAPeerHeldBackFromNewReplicas(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{seed(peer)}
	v, schedule, replicas, shortfall := openShortfall(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)
	shortfall.eligibility = peersHeldBack{peer: {}}

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, peer, fakePosting(word, url))

	due, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due.Offers) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	if seeds := due.Offers[0].Seeds; len(seeds) != 1 || seeds[0].Hash != peer {
		t.Fatalf("Seeds = %v, want the held-back peer that holds the replica", seeds)
	}
}

func TestDueOwesNoReplicasWhenNoPeerIsCloserThanThisNode(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	v, schedule, _, shortfall := openShortfall(
		t, func() time.Time { return now }, postings, fakeReachability{},
	)

	store(t, v, schedule, word, url)

	due, err := shortfall.Due(context.Background(), 10, nil)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due.Offers) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	missing := due.Offers[0]
	if missing.ReplicasNeeded != 0 || len(missing.Seeds) != 0 {
		t.Fatalf(
			"missing = %+v, want no replicas owed: this node holds the only one the DHT asks for",
			missing,
		)
	}
}

func TestDueLowersReplicasOwedWhenThisNodeIsResponsible(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{
		seed(yacymodel.WordHash("p1")),
		seed(yacymodel.WordHash("p2")),
		seed(yacymodel.WordHash("p3")),
	}
	v, schedule, _, shortfall := openShortfall(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)
	shortfall.redundancy = 3
	shortfall.self = thisNodeCloserThanEveryPeer()

	store(t, v, schedule, word, url)

	due, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due.Offers) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	missing := due.Offers[0]
	if missing.ReplicasNeeded != 2 || len(missing.Seeds) != 2 {
		t.Fatalf(
			"missing = %+v, want two replicas owed: this node is one of the three the DHT"+
				" makes responsible",
			missing,
		)
	}
}

func TestDueLeavesReplicasOwedWhenThisNodeIsOutsideTheWindow(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{
		seed(yacymodel.WordHash("p1")),
		seed(yacymodel.WordHash("p2")),
		seed(yacymodel.WordHash("p3")),
	}
	v, schedule, _, shortfall := openShortfall(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)
	shortfall.redundancy = 3

	store(t, v, schedule, word, url)

	due, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due.Offers) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	missing := due.Offers[0]
	if missing.ReplicasNeeded != 3 || len(missing.Seeds) != 3 {
		t.Fatalf(
			"missing = %+v, want three replicas owed: this node is farther than every peer",
			missing,
		)
	}
}

func TestDueNarrowsTheLedgerWindowWhenThisNodeIsResponsible(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{
		seed(yacymodel.WordHash("p1")),
		seed(yacymodel.WordHash("p2")),
	}
	v, schedule, replicas, shortfall := openShortfall(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)
	shortfall.redundancy = 2
	shortfall.self = thisNodeCloserThanEveryPeer()

	store(t, v, schedule, word, url)
	for _, peer := range peers {
		recordAccepted(t, v, replicas, peer.Hash, fakePosting(word, url))
	}

	due, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due.Stale) != 1 || len(due.Stale[0].Peers) != 1 {
		t.Fatalf(
			"due.Stale = %+v, want the one holder beyond the single replica peers owe",
			due.Stale,
		)
	}
}

func TestDueCollectsGoneForRemovedPosting(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	peer := yacymodel.WordHash("peer")
	v, schedule, _, shortfall := openShortfall(
		t, func() time.Time { return now }, nil, fakeReachability{},
	)
	peers := []yacymodel.Seed{seed(peer)}

	store(t, v, schedule, word, url)

	due, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due.Offers) != 0 || len(due.Gone) != 1 || due.Gone[0].Word != word {
		t.Fatalf("due = %+v, want a single gone entry for %v", due, word)
	}
}

func TestDueReportsStaleReplicaWithoutDroppingIt(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	stalePeer, fresh := yacymodel.WordHash("stale"), yacymodel.WordHash("fresh")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	v, schedule, replicas, shortfall := openShortfall(
		t, func() time.Time { return now }, postings, fakeReachability{},
	)
	peers := []yacymodel.Seed{seed(fresh)}

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, stalePeer, fakePosting(word, url))

	due, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due.Offers) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	missing := due.Offers[0]
	if len(missing.Seeds) != 1 || missing.Seeds[0].Hash != fresh {
		t.Fatalf(
			"missing = %+v, want an offer to %v regardless of stale peer %v",
			missing, fresh, stalePeer,
		)
	}
	if len(due.Stale) != 1 || len(due.Stale[0].Peers) != 1 ||
		due.Stale[0].Peers[0] != stalePeer {
		t.Fatalf("due.Stale = %+v, want the one stale peer %v", due.Stale, stalePeer)
	}
	stored, err := replicas.Holders(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Holders: %v", err)
	}
	if len(stored) != 1 || stored[0] != stalePeer {
		t.Fatalf("replicas = %v, want [%v] unchanged by Due", stored, stalePeer)
	}
}

func TestDueOffersOnlyAsManyReplicasAsAreNeeded(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	absentHolder := yacymodel.WordHash("absent")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	v, schedule, replicas, shortfall := openShortfall(
		t, func() time.Time { return now }, postings,
		fakeReachability{recentlyReachable: map[yacymodel.Hash]struct{}{absentHolder: {}}},
	)
	shortfall.redundancy = 3

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, absentHolder, fakePosting(word, url))

	due, err := shortfall.Due(context.Background(), 10, []yacymodel.Seed{
		seed(yacymodel.WordHash("p1")),
		seed(yacymodel.WordHash("p2")),
		seed(yacymodel.WordHash("p3")),
	})
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due.Offers) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	missing := due.Offers[0]
	if missing.ReplicasNeeded != 2 {
		t.Fatalf(
			"ReplicasNeeded = %d, want 2 with one credible holder of three",
			missing.ReplicasNeeded,
		)
	}
	if len(missing.Seeds) != missing.ReplicasNeeded {
		t.Fatalf(
			"Seeds = %d seeds, want %d so accepting every offer stays within redundancy",
			len(missing.Seeds),
			missing.ReplicasNeeded,
		)
	}
}

func TestDuePrunesReachableHolderDisplacedByCloserPeer(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	first, second := yacymodel.WordHash("first"), yacymodel.WordHash("second")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{seed(first), seed(second)}
	v, schedule, replicas, shortfall := openShortfall(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)

	store(t, v, schedule, word, url)
	for _, peer := range peers {
		recordAccepted(t, v, replicas, peer.Hash, fakePosting(word, url))
	}

	due, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due.Offers) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	if len(due.Stale) != 1 || len(due.Stale[0].Peers) != 1 {
		t.Fatalf(
			"due.Stale = %+v, want the one holder beyond a redundancy of %d",
			due.Stale, shortfallRedundancy,
		)
	}
}

func TestDueKeepsHolderThatStoppedAcceptingIndex(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	v, schedule, replicas, shortfall := openShortfall(
		t, func() time.Time { return now }, postings,
		fakeReachability{reachable: []yacymodel.Seed{indexDecliningSeed(peer)}},
	)

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, peer, fakePosting(word, url))

	due, err := shortfall.Due(
		context.Background(), 10, []yacymodel.Seed{indexDecliningSeed(peer)},
	)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due.Offers) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	missing := due.Offers[0]
	if len(due.Stale) != 0 {
		t.Fatalf(
			"due.Stale = %+v, want none: a peer that stops accepting a remote index "+
				"still serves the replica it holds",
			due.Stale,
		)
	}
	if missing.ReplicasNeeded != 0 || len(missing.Seeds) != 0 {
		t.Fatalf(
			"missing = %+v, want no further replicas and no offer to the holder",
			missing,
		)
	}
}

func TestDueOffersToNextPeerWhenClosestIsHeldBack(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{
		seed(yacymodel.WordHash("p1")),
		seed(yacymodel.WordHash("p2")),
	}
	v, schedule, _, shortfall := openShortfall(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)

	store(t, v, schedule, word, url)

	first, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(first.Offers) != 1 || len(first.Offers[0].Seeds) != 1 {
		t.Fatalf("due = %+v, want one posting offered to the closest peer", first)
	}
	closest := first.Offers[0].Seeds[0].Hash

	shortfall.eligibility = peersHeldBack{closest: {}}

	second, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(second.Offers) != 1 || len(second.Offers[0].Seeds) != 1 {
		t.Fatalf("due = %+v, want one posting still offered somewhere", second)
	}
	if next := second.Offers[0].Seeds[0].Hash; next == closest {
		t.Fatalf(
			"Seeds[0] = %v, want a peer other than the held-back closest peer %v",
			next, closest,
		)
	}
}

func TestDueKeepsHolderOnHeldBackPeer(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{seed(peer)}
	v, schedule, replicas, shortfall := openShortfall(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)
	shortfall.eligibility = peersHeldBack{peer: {}}

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, peer, fakePosting(word, url))

	due, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due.Stale) != 0 {
		t.Fatalf(
			"due.Stale = %+v, want none: a peer in cooldown still serves the replica it holds",
			due.Stale,
		)
	}
	if len(due.Offers) != 1 || due.Offers[0].ReplicasNeeded != 0 {
		t.Fatalf("due.Offers = %+v, want no further replicas needed", due.Offers)
	}
}

func TestDueNarrowsResponsibilityWindowToPeersInCooldown(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{
		seed(yacymodel.WordHash("p1")),
		seed(yacymodel.WordHash("p2")),
		seed(yacymodel.WordHash("p3")),
	}
	v, schedule, replicas, shortfall := openShortfall(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)

	store(t, v, schedule, word, url)

	ranked, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	shortfall.eligibility = peersHeldBack{ranked.Offers[0].Seeds[0].Hash: {}}

	eligible, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due with the closest peer in cooldown: %v", err)
	}
	recordAccepted(t, v, replicas, eligible.Offers[0].Seeds[0].Hash, fakePosting(word, url))

	due, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due after cooldown: %v", err)
	}
	if len(due.Offers) != 1 || due.Offers[0].ReplicasNeeded != 1 {
		t.Fatalf(
			"due.Offers = %+v, want one replica needed: a peer in cooldown does not widen"+
				" the responsibility window",
			due.Offers,
		)
	}
	if len(due.Stale) != 0 {
		t.Fatalf("due.Stale = %+v, want none while the posting is below redundancy", due.Stale)
	}
}

func TestDueKeepsHolderOutsideWindowUntilRedundancyIsMet(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{
		seed(yacymodel.WordHash("p1")),
		seed(yacymodel.WordHash("p2")),
	}
	v, schedule, replicas, shortfall := openShortfall(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)

	store(t, v, schedule, word, url)

	ranked, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	closest := ranked.Offers[0].Seeds[0].Hash
	outside := otherThan(peers, closest)

	recordAccepted(t, v, replicas, outside, fakePosting(word, url))

	kept, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due below redundancy: %v", err)
	}
	if len(kept.Stale) != 0 {
		t.Fatalf(
			"due.Stale = %+v, want none: a reachable holder outside the window still serves it",
			kept.Stale,
		)
	}

	recordAccepted(t, v, replicas, closest, fakePosting(word, url))

	dropped, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due at redundancy: %v", err)
	}
	if len(dropped.Stale) != 1 || dropped.Stale[0].Peers[0] != outside {
		t.Fatalf("due.Stale = %+v, want [%v] once redundancy is met", dropped.Stale, outside)
	}
}

func TestDueDropsUnreachableHolderBelowRedundancy(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	ghost := yacymodel.WordHash("ghost")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{seed(yacymodel.WordHash("p1"))}
	v, schedule, replicas, shortfall := openShortfall(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, ghost, fakePosting(word, url))

	due, err := shortfall.Due(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due.Stale) != 1 || due.Stale[0].Peers[0] != ghost {
		t.Fatalf("due.Stale = %+v, want [%v]: the peer is gone", due.Stale, ghost)
	}
	if len(due.Offers) != 1 || due.Offers[0].ReplicasNeeded != 1 {
		t.Fatalf("due.Offers = %+v, want one replica needed", due.Offers)
	}
}

func otherThan(peers []yacymodel.Seed, closest yacymodel.Hash) yacymodel.Hash {
	for _, peer := range peers {
		if peer.Hash != closest {
			return peer.Hash
		}
	}

	panic("no peer other than the closest")
}
