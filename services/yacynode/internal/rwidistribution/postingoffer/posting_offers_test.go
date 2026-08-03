package postingoffer

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type fakePostingIndex struct {
	postings map[yacymodel.Hash]yacymodel.RWIPosting
	unread   error
}

func (f fakePostingIndex) RWICount(context.Context) (int, error) { return len(f.postings), nil }

func (f fakePostingIndex) PostingOf(
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

func indexDecliningSeed(hash yacymodel.Hash) yacymodel.Seed {
	declining := seed(hash)
	declining.Capabilities = yacymodel.Some(yacymodel.PeerCapabilities{AcceptRemoteIndex: false})

	return declining
}

type everyPeerEligible struct{}

func (everyPeerEligible) EligiblePeers(peers []yacymodel.Seed) []yacymodel.Seed { return peers }

type peersHeldBack map[yacymodel.Hash]struct{}

func (p peersHeldBack) EligiblePeers(peers []yacymodel.Seed) []yacymodel.Seed {
	eligible := make([]yacymodel.Seed, 0, len(peers))
	for _, peer := range peers {
		if _, held := p[peer.Hash]; !held {
			eligible = append(eligible, peer)
		}
	}

	return eligible
}

const postingOffersRedundancy = 1

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

func openPostingOffers(
	t *testing.T,
	now func() time.Time,
	postings map[yacymodel.Hash]yacymodel.RWIPosting,
	reachability Reachability,
) (*vault.Vault, *postingofferschedule.Schedule, *postingreplicas.Replicas, *PostingOffers) {
	t.Helper()

	return openPostingOffersReportingTo(
		t, now, postings, reachability, discardedRingSectorObservations{},
	)
}

func openPostingOffersReportingTo(
	t *testing.T,
	now func() time.Time,
	postings map[yacymodel.Hash]yacymodel.RWIPosting,
	reachability Reachability,
	observer Observer,
) (*vault.Vault, *postingofferschedule.Schedule, *postingreplicas.Replicas, *PostingOffers) {
	t.Helper()

	v, err := memvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	schedule, err := postingofferschedule.Open(v, now, discardedScheduleObservations{})
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

	postingOffers := New(
		v,
		schedule,
		replicas,
		fakePostingIndex{postings: postings},
		reachability,
		everyPeerEligible{},
		observer,
		partitions,
		thisNodeFartherThanEveryPeer(),
		postingOffersRedundancy,
	)

	return v, schedule, replicas, postingOffers
}

func store(
	t *testing.T,
	v *vault.Vault,
	schedule *postingofferschedule.Schedule,
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
	v, schedule, _, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, postings, fakeReachability{},
	)
	peers := []yacymodel.Seed{seed(peer)}

	store(t, v, schedule, word, url)

	due, dueGone, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due) != 1 || len(dueGone) != 0 {
		t.Fatalf("due = %+v, want a single posting and no gone entries", due)
	}
	missing := due[0]
	if missing.AcceptancesNeeded != 1 || len(missing.Peers) != 1 ||
		missing.Peers[0].Hash != peer {
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
	v, schedule, replicas, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, peer, fakePosting(word, url))

	due, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	offer := due[0]
	if offer.AcceptancesNeeded != 0 {
		t.Fatalf("AcceptancesNeeded = %d, want zero", offer.AcceptancesNeeded)
	}
	if len(offer.Peers) != 1 || offer.Peers[0].Hash != peer {
		t.Fatalf("Peers = %v, want the peer that holds the replica", offer.Peers)
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
	v, schedule, replicas, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)
	postingOffers.eligibility = peersHeldBack{peer: {}}

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, peer, fakePosting(word, url))

	due, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	if recipients := due[0].Peers; len(recipients) != 1 ||
		recipients[0].Hash != peer {
		t.Fatalf("Peers = %v, want the held-back peer that holds the replica", recipients)
	}
}

func TestDueOwesNoReplicasWhenNoPeerIsCloserThanThisNode(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	v, schedule, _, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, postings, fakeReachability{},
	)

	store(t, v, schedule, word, url)

	due, _, err := postingOffers.DueNow(context.Background(), 10, nil)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	missing := due[0]
	if missing.AcceptancesNeeded != 0 || len(missing.Peers) != 0 {
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
	v, schedule, _, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)
	postingOffers.redundancy = 3
	postingOffers.self = thisNodeCloserThanEveryPeer()

	store(t, v, schedule, word, url)

	due, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	missing := due[0]
	if missing.AcceptancesNeeded != 2 || len(missing.Peers) != 2 {
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
	v, schedule, _, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)
	postingOffers.redundancy = 3

	store(t, v, schedule, word, url)

	due, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	missing := due[0]
	if missing.AcceptancesNeeded != 3 || len(missing.Peers) != 3 {
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
	v, schedule, replicas, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)
	postingOffers.redundancy = 2
	postingOffers.self = thisNodeCloserThanEveryPeer()

	store(t, v, schedule, word, url)
	for _, peer := range peers {
		recordAccepted(t, v, replicas, peer.Hash, fakePosting(word, url))
	}

	due, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due) != 1 || len(due[0].StaleHolders) != 1 {
		t.Fatalf(
			"due = %+v, want the one holder beyond the single replica peers owe",
			due,
		)
	}
}

func TestDueCollectsGoneForRemovedPosting(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	peer := yacymodel.WordHash("peer")
	v, schedule, _, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, nil, fakeReachability{},
	)
	peers := []yacymodel.Seed{seed(peer)}

	store(t, v, schedule, word, url)

	due, dueGone, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due) != 0 || len(dueGone) != 1 || dueGone[0].Word != word {
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
	v, schedule, replicas, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, postings, fakeReachability{},
	)
	peers := []yacymodel.Seed{seed(fresh)}

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, stalePeer, fakePosting(word, url))

	due, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	missing := due[0]
	if len(missing.Peers) != 1 || missing.Peers[0].Hash != fresh {
		t.Fatalf(
			"missing = %+v, want an offer to %v regardless of stale peer %v",
			missing, fresh, stalePeer,
		)
	}
	if len(missing.StaleHolders) != 1 || missing.StaleHolders[0] != stalePeer {
		t.Fatalf("StaleHolders = %+v, want the one stale peer %v", missing.StaleHolders, stalePeer)
	}
	stored := holdersOf(t, v, replicas, word, url)
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
	v, schedule, replicas, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, postings,
		fakeReachability{recentlyReachable: map[yacymodel.Hash]struct{}{absentHolder: {}}},
	)
	postingOffers.redundancy = 3

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, absentHolder, fakePosting(word, url))

	due, _, err := postingOffers.DueNow(context.Background(), 10, []yacymodel.Seed{
		seed(yacymodel.WordHash("p1")),
		seed(yacymodel.WordHash("p2")),
		seed(yacymodel.WordHash("p3")),
	})
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	missing := due[0]
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

func TestDuePrunesReachableHolderDisplacedByCloserPeer(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	first, second := yacymodel.WordHash("first"), yacymodel.WordHash("second")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{seed(first), seed(second)}
	v, schedule, replicas, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)

	store(t, v, schedule, word, url)
	for _, peer := range peers {
		recordAccepted(t, v, replicas, peer.Hash, fakePosting(word, url))
	}

	due, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due) != 1 || len(due[0].StaleHolders) != 1 {
		t.Fatalf(
			"due = %+v, want the one holder beyond a redundancy of %d",
			due, postingOffersRedundancy,
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
	v, schedule, replicas, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, postings,
		fakeReachability{reachable: []yacymodel.Seed{indexDecliningSeed(peer)}},
	)

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, peer, fakePosting(word, url))

	due, _, err := postingOffers.DueNow(
		context.Background(), 10, []yacymodel.Seed{indexDecliningSeed(peer)},
	)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	missing := due[0]
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
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{
		seed(yacymodel.WordHash("p1")),
		seed(yacymodel.WordHash("p2")),
	}
	v, schedule, _, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)

	store(t, v, schedule, word, url)

	first, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(first) != 1 || len(first[0].Peers) != 1 {
		t.Fatalf("due = %+v, want one posting offered to the closest peer", first)
	}
	closest := first[0].Peers[0].Hash

	postingOffers.eligibility = peersHeldBack{closest: {}}

	second, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(second) != 1 || len(second[0].Peers) != 1 {
		t.Fatalf("due = %+v, want one posting still offered somewhere", second)
	}
	if next := second[0].Peers[0].Hash; next == closest {
		t.Fatalf(
			"Recipients[0] = %v, want a peer other than the held-back closest peer %v",
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
	v, schedule, replicas, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)
	postingOffers.eligibility = peersHeldBack{peer: {}}

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, peer, fakePosting(word, url))

	due, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due) != 1 || len(due[0].StaleHolders) != 0 {
		t.Fatalf(
			"due = %+v, want none stale: a peer in cooldown still serves the replica"+
				" it holds",
			due,
		)
	}
	if due[0].AcceptancesNeeded != 0 {
		t.Fatalf("due = %+v, want no further replicas needed", due)
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
	v, schedule, replicas, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)

	store(t, v, schedule, word, url)

	ranked, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	postingOffers.eligibility = peersHeldBack{ranked[0].Peers[0].Hash: {}}

	eligible, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due with the closest peer in cooldown: %v", err)
	}
	recordAccepted(t, v, replicas, eligible[0].Peers[0].Hash, fakePosting(word, url))

	due, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due after cooldown: %v", err)
	}
	if len(due) != 1 || due[0].AcceptancesNeeded != 1 {
		t.Fatalf(
			"due = %+v, want one replica needed: a peer in cooldown does not widen"+
				" the responsibility window",
			due,
		)
	}
	if len(due[0].StaleHolders) != 0 {
		t.Fatalf(
			"StaleHolders = %+v, want none while the posting is below redundancy",
			due[0].StaleHolders,
		)
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
	v, schedule, replicas, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)

	store(t, v, schedule, word, url)

	ranked, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	closest := ranked[0].Peers[0].Hash
	outside := otherThan(peers, closest)

	recordAccepted(t, v, replicas, outside, fakePosting(word, url))

	kept, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due below redundancy: %v", err)
	}
	if len(kept[0].StaleHolders) != 0 {
		t.Fatalf(
			"StaleHolders = %+v, want none: a reachable holder outside the window still serves it",
			kept[0].StaleHolders,
		)
	}

	recordAccepted(t, v, replicas, closest, fakePosting(word, url))

	dropped, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due at redundancy: %v", err)
	}
	if len(dropped[0].StaleHolders) != 1 || dropped[0].StaleHolders[0] != outside {
		t.Fatalf(
			"StaleHolders = %+v, want [%v] once redundancy is met",
			dropped[0].StaleHolders, outside,
		)
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
	v, schedule, replicas, postingOffers := openPostingOffers(
		t, func() time.Time { return now }, postings, fakeReachability{reachable: peers},
	)

	store(t, v, schedule, word, url)
	recordAccepted(t, v, replicas, ghost, fakePosting(word, url))

	due, _, err := postingOffers.DueNow(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due[0].StaleHolders) != 1 || due[0].StaleHolders[0] != ghost {
		t.Fatalf("StaleHolders = %+v, want [%v]: the peer is gone", due[0].StaleHolders, ghost)
	}
	if due[0].AcceptancesNeeded != 1 {
		t.Fatalf("due = %+v, want one replica needed", due)
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

func holdersOf(
	t *testing.T,
	v *vault.Vault,
	replicas *postingreplicas.Replicas,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) []yacymodel.Hash {
	t.Helper()

	var holders []yacymodel.Hash
	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		holders, err = replicas.HoldersOf(tx, postingidentity.IdentityOf(word, url))

		return err
	}); err != nil {
		t.Fatalf("Holders: %v", err)
	}

	return holders
}

type discardedScheduleObservations struct{}

func (discardedScheduleObservations) ObserveScheduledPostings(int) {}

func (discardedScheduleObservations) ObserveLongestOfferLateness(time.Duration) {}

func TestDueReportsAcceptingPeersPerRingSector(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	observer := &recordedRingSectorObservations{}
	v, schedule, _, postingOffers := openPostingOffersReportingTo(
		t, func() time.Time { return now }, postings, fakeReachability{}, observer,
	)
	peers := []yacymodel.Seed{seed(peer), indexDecliningSeed(yacymodel.WordHash("declining"))}

	store(t, v, schedule, word, url)

	if _, _, err := postingOffers.DueNow(context.Background(), 10, peers); err != nil {
		t.Fatalf("DueNow: %v", err)
	}

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
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash()
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{seed(peer)}
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}
	wantPosition := yacymodel.PostingPosition(word, url, partitions)

	for name, reachability := range map[string]fakeReachability{
		"replica needed": {},
		"replica held":   {reachable: peers},
	} {
		t.Run(name, func(t *testing.T) {
			v, schedule, replicas, postingOffers := openPostingOffers(
				t, func() time.Time { return now }, postings, reachability,
			)
			store(t, v, schedule, word, url)
			if len(reachability.reachable) > 0 {
				recordAccepted(t, v, replicas, peer, fakePosting(word, url))
			}

			due, _, err := postingOffers.DueNow(context.Background(), 10, peers)
			if err != nil {
				t.Fatalf("DueNow: %v", err)
			}
			if len(due) != 1 {
				t.Fatalf("due = %+v, want a single posting", due)
			}
			if due[0].PostingPosition != wantPosition {
				t.Errorf(
					"PostingPosition = %d, want %d", due[0].PostingPosition, wantPosition,
				)
			}
		})
	}
}
