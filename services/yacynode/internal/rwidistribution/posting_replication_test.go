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
	responsible []yacymodel.Seed
	reachable   []yacymodel.Seed
}

func (fakeRoster) Discover(context.Context, ...yacymodel.Seed)            {}
func (fakeRoster) ConfirmReachable(context.Context, yacymodel.Hash)       {}
func (fakeRoster) ConfirmUnreachable(context.Context, yacymodel.Hash)     {}
func (fakeRoster) UnreachablePeers(context.Context, int) []yacymodel.Seed { return nil }

func (f fakeRoster) ReachablePeers(context.Context) []yacymodel.Seed {
	return f.reachable
}

func (f fakeRoster) PeersResponsibleFor(
	context.Context,
	yacymodel.DHTPosition,
	int,
) []yacymodel.Seed {
	return f.responsible
}

func seed(hash yacymodel.Hash) yacymodel.Seed {
	return yacymodel.Seed{Hash: hash}
}

const postingReplicationReaderRedundancy = 1

func openPostingReplicationReader(
	t *testing.T,
	now func() time.Time,
	postings map[yacymodel.Hash]yacymodel.RWIPosting,
	responsible []yacymodel.Seed,
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
		schedule:   schedule,
		ledger:     ledger,
		postings:   fakePostingIndex{postings: postings},
		roster:     fakeRoster{responsible: responsible, reachable: responsible},
		partitions: partitions,
		redundancy: postingReplicationReaderRedundancy,
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
		[]yacymodel.Seed{seed(peer)},
	)

	store(t, schedule, word, url)

	due, err := reader.DueReplication(context.Background(), 10)
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
	schedule, ledger, reader := openPostingReplicationReader(
		t,
		func() time.Time { return now },
		postings,
		[]yacymodel.Seed{seed(peer)},
	)

	store(t, schedule, word, url)
	if err := ledger.RecordAccepted(
		context.Background(),
		postingOffer{Peer: seed(peer), Postings: []yacymodel.RWIPosting{fakePosting(word, url)}},
	); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	due, err := reader.DueReplication(context.Background(), 10)
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
		nil,
	)

	store(t, schedule, word, url)

	due, err := reader.DueReplication(context.Background(), 10)
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
		[]yacymodel.Seed{seed(peer)},
	)

	store(t, schedule, word, url)

	due, err := reader.DueReplication(context.Background(), 10)
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
		[]yacymodel.Seed{seed(fresh)},
	)

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

	due, err := reader.DueReplication(context.Background(), 10)
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
