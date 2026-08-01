package rwidistribution

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
)

type fakePostingIndex struct {
	postings map[yacymodel.Hash]yacymodel.RWIPosting
}

func (f fakePostingIndex) RWICount(context.Context) (int, error) { return len(f.postings), nil }

func (f fakePostingIndex) Posting(
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

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}

func fakePosting(word yacymodel.Hash, url yacymodel.URLHash) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{WordHash: word, URLHash: url}
}

type fakeRoster struct {
	reachable         []yacymodel.Seed
	recentlyReachable map[yacymodel.Hash]struct{}
}

func (fakeRoster) Discover(context.Context, ...yacymodel.Seed)            {}
func (fakeRoster) ConfirmReachable(context.Context, yacymodel.Hash)       {}
func (fakeRoster) ConfirmUnreachable(context.Context, yacymodel.Hash)     {}
func (fakeRoster) UnreachablePeers(context.Context, int) []yacymodel.Seed { return nil }

func (f fakeRoster) ReachablePeers(context.Context) []yacymodel.Seed {
	return f.reachable
}

func (f fakeRoster) Reachable(_ context.Context, peer yacymodel.Hash) bool {
	for _, seed := range f.reachable {
		if seed.Hash == peer {
			return true
		}
	}

	return false
}

func (f fakeRoster) RecentlyReachable(_ context.Context, peer yacymodel.Hash) bool {
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

const postingReplicationReaderRedundancy = 1

func openPostingReplicationReader(
	t *testing.T,
	now func() time.Time,
	postings map[yacymodel.Hash]yacymodel.RWIPosting,
	reachability peerReachability,
) (*postingOfferSchedule, *replicaLedger, *postingReplicationReader) {
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

	schedule, err := openPostingOfferSchedule(v, now)
	if err != nil {
		t.Fatalf("openPostingOfferSchedule: %v", err)
	}
	ledger, err := openReplicaLedger(v, schedule)
	if err != nil {
		t.Fatalf("openReplicaLedger: %v", err)
	}
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}

	reader := &postingReplicationReader{
		schedule:     schedule,
		ledger:       ledger,
		postings:     fakePostingIndex{postings: postings},
		reachability: reachability,
		partitions:   partitions,
		redundancy:   postingReplicationReaderRedundancy,
	}

	return schedule, ledger, reader
}

func TestDueReplicationReturnsMissingCopyForDuePosting(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, _, reader := openPostingReplicationReader(
		t,
		func() time.Time { return now },
		postings,
		fakeRoster{},
	)
	peers := []yacymodel.Seed{seed(peer)}

	store(t, schedule, word, url)

	due, err := reader.DueReplication(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("DueReplication: %v", err)
	}
	if len(due.Postings) != 1 || len(due.Gone) != 0 {
		t.Fatalf("due = %+v, want a single posting and no gone entries", due)
	}
	replication := due.Postings[0]
	if replication.CopiesNeeded != 1 || len(replication.SeedsMissingCopy) != 1 ||
		replication.SeedsMissingCopy[0].Hash != peer {
		t.Fatalf("replication = %+v, want one copy needed from %v", replication, peer)
	}
}

func TestDueReplicationReportsNoCopiesNeededWhenReplicated(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{seed(peer)}
	schedule, ledger, reader := openPostingReplicationReader(
		t,
		func() time.Time { return now },
		postings,
		fakeRoster{reachable: peers},
	)

	store(t, schedule, word, url)
	if err := ledger.RecordAccepted(
		context.Background(),
		postingOffer{Peer: seed(peer), Postings: []yacymodel.RWIPosting{fakePosting(word, url)}},
	); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	due, err := reader.DueReplication(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("DueReplication: %v", err)
	}
	if len(due.Postings) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	replication := due.Postings[0]
	if replication.CopiesNeeded != 0 || len(replication.SeedsMissingCopy) != 0 {
		t.Fatalf("replication = %+v, want zero copies needed", replication)
	}
}

func TestDueReplicationLeavesCopiesNeededWithNoResponsiblePeers(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, _, reader := openPostingReplicationReader(
		t,
		func() time.Time { return now },
		postings,
		fakeRoster{},
	)

	store(t, schedule, word, url)

	due, err := reader.DueReplication(context.Background(), 10, nil)
	if err != nil {
		t.Fatalf("DueReplication: %v", err)
	}
	if len(due.Postings) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	replication := due.Postings[0]
	if replication.CopiesNeeded != 1 || len(replication.SeedsMissingCopy) != 0 {
		t.Fatalf(
			"replication = %+v, want one copy needed with no peers to offer it to",
			replication,
		)
	}
}

func TestDueReplicationCollectsGoneForRemovedPosting(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	schedule, _, reader := openPostingReplicationReader(
		t,
		func() time.Time { return now },
		nil,
		fakeRoster{},
	)
	peers := []yacymodel.Seed{seed(peer)}

	store(t, schedule, word, url)

	due, err := reader.DueReplication(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("DueReplication: %v", err)
	}
	if len(due.Postings) != 0 || len(due.Gone) != 1 || due.Gone[0].Word != word {
		t.Fatalf("due = %+v, want a single gone entry for %v", due, word)
	}
}

func TestDueReplicationReportsStaleReplicaWithoutDroppingIt(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	stalePeer, fresh := yacymodel.WordHash("stale"), yacymodel.WordHash("fresh")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, ledger, reader := openPostingReplicationReader(
		t,
		func() time.Time { return now },
		postings,
		fakeRoster{},
	)
	peers := []yacymodel.Seed{seed(fresh)}

	store(t, schedule, word, url)
	if err := ledger.RecordAccepted(
		context.Background(),
		postingOffer{
			Peer:     seed(stalePeer),
			Postings: []yacymodel.RWIPosting{fakePosting(word, url)},
		},
	); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	due, err := reader.DueReplication(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("DueReplication: %v", err)
	}
	if len(due.Postings) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	replication := due.Postings[0]
	if len(replication.SeedsMissingCopy) != 1 || replication.SeedsMissingCopy[0].Hash != fresh {
		t.Fatalf(
			"replication = %+v, want an offer to %v regardless of stale peer %v",
			replication, fresh, stalePeer,
		)
	}
	if len(replication.PeerHashesNoLongerResponsible) != 1 ||
		replication.PeerHashesNoLongerResponsible[0] != stalePeer {
		t.Fatalf(
			"replication.PeerHashesNoLongerResponsible = %v, want [%v]",
			replication.PeerHashesNoLongerResponsible, stalePeer,
		)
	}
	replicas, err := ledger.Replicas(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	if len(replicas) != 1 || replicas[0] != stalePeer {
		t.Fatalf("replicas = %v, want [%v] unchanged by DueReplication", replicas, stalePeer)
	}
}

func TestDueReplicationOffersOnlyAsManyCopiesAsAreNeeded(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	absentHolder := yacymodel.WordHash("absent")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, ledger, reader := openPostingReplicationReader(
		t,
		func() time.Time { return now },
		postings,
		fakeRoster{recentlyReachable: map[yacymodel.Hash]struct{}{absentHolder: {}}},
	)
	reader.redundancy = 3

	store(t, schedule, word, url)
	if err := ledger.RecordAccepted(
		context.Background(),
		postingOffer{
			Peer:     seed(absentHolder),
			Postings: []yacymodel.RWIPosting{fakePosting(word, url)},
		},
	); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	due, err := reader.DueReplication(context.Background(), 10, []yacymodel.Seed{
		seed(yacymodel.WordHash("p1")),
		seed(yacymodel.WordHash("p2")),
		seed(yacymodel.WordHash("p3")),
	})
	if err != nil {
		t.Fatalf("DueReplication: %v", err)
	}
	if len(due.Postings) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	replication := due.Postings[0]
	if replication.CopiesNeeded != 2 {
		t.Fatalf(
			"CopiesNeeded = %d, want 2 with one credible holder of three",
			replication.CopiesNeeded,
		)
	}
	if len(replication.SeedsMissingCopy) != replication.CopiesNeeded {
		t.Fatalf(
			"SeedsMissingCopy = %d seeds, want %d so accepting every offer stays within redundancy",
			len(replication.SeedsMissingCopy),
			replication.CopiesNeeded,
		)
	}
}

func TestDueReplicationPrunesReachableHolderDisplacedByCloserPeer(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	first, second := yacymodel.WordHash("first"), yacymodel.WordHash("second")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	peers := []yacymodel.Seed{seed(first), seed(second)}
	schedule, ledger, reader := openPostingReplicationReader(
		t,
		func() time.Time { return now },
		postings,
		fakeRoster{reachable: peers},
	)

	store(t, schedule, word, url)
	for _, peer := range peers {
		if err := ledger.RecordAccepted(
			context.Background(),
			postingOffer{Peer: peer, Postings: []yacymodel.RWIPosting{fakePosting(word, url)}},
		); err != nil {
			t.Fatalf("RecordAccepted: %v", err)
		}
	}

	due, err := reader.DueReplication(context.Background(), 10, peers)
	if err != nil {
		t.Fatalf("DueReplication: %v", err)
	}
	if len(due.Postings) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	if got := due.Postings[0].PeerHashesNoLongerResponsible; len(got) != 1 {
		t.Fatalf(
			"PeerHashesNoLongerResponsible = %v, want the one holder beyond a redundancy of %d",
			got, postingReplicationReaderRedundancy,
		)
	}
}

func TestDueReplicationKeepsHolderThatStoppedAcceptingIndex(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, ledger, reader := openPostingReplicationReader(
		t,
		func() time.Time { return now },
		postings,
		fakeRoster{reachable: []yacymodel.Seed{indexDecliningSeed(peer)}},
	)

	store(t, schedule, word, url)
	if err := ledger.RecordAccepted(
		context.Background(),
		postingOffer{Peer: seed(peer), Postings: []yacymodel.RWIPosting{fakePosting(word, url)}},
	); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	due, err := reader.DueReplication(
		context.Background(), 10, []yacymodel.Seed{indexDecliningSeed(peer)},
	)
	if err != nil {
		t.Fatalf("DueReplication: %v", err)
	}
	if len(due.Postings) != 1 {
		t.Fatalf("due = %+v, want a single posting", due)
	}
	replication := due.Postings[0]
	if len(replication.PeerHashesNoLongerResponsible) != 0 {
		t.Fatalf(
			"replication.PeerHashesNoLongerResponsible = %v, want none: a peer that stops "+
				"accepting a remote index still serves the copy it holds",
			replication.PeerHashesNoLongerResponsible,
		)
	}
	if replication.CopiesNeeded != 0 || len(replication.SeedsMissingCopy) != 0 {
		t.Fatalf(
			"replication = %+v, want no further copies and no offer to the holder",
			replication,
		)
	}
}
