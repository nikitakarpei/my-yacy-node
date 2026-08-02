package rwidistribution

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributioncycle"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}

type fakePostingIndex struct{}

func (fakePostingIndex) RWICount(context.Context) (int, error) { return 0, nil }

func (fakePostingIndex) Posting(
	context.Context, yacymodel.Hash, yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	return yacymodel.RWIPosting{}, false, nil
}

func (fakePostingIndex) ScanWord(
	context.Context, yacymodel.Hash, func(yacymodel.RWIPosting) (bool, error),
) error {
	return nil
}

type fakeRoster struct{}

func (fakeRoster) Discover(context.Context, ...yacymodel.Seed)            {}
func (fakeRoster) ConfirmReachable(context.Context, yacymodel.Hash)       {}
func (fakeRoster) ConfirmUnreachable(context.Context, yacymodel.Hash)     {}
func (fakeRoster) UnreachablePeers(context.Context, int) []yacymodel.Seed { return nil }
func (fakeRoster) ReachablePeers(context.Context) []yacymodel.Seed        { return nil }
func (fakeRoster) Reachable(context.Context, yacymodel.Hash) bool         { return false }
func (fakeRoster) RecentlyReachable(context.Context, yacymodel.Hash) bool { return false }

type fakeURLDirectory struct{}

func (fakeURLDirectory) MetadataByHash(
	context.Context, []yacymodel.URLHash,
) ([]yacymodel.URLMetadata, error) {
	return nil, nil
}

func (fakeURLDirectory) MissingURLs(
	context.Context, []yacymodel.URLHash,
) ([]yacymodel.URLHash, error) {
	return nil, nil
}

func (fakeURLDirectory) Count(context.Context) (int, error) { return 0, nil }

func openDistribution(t *testing.T, now func() time.Time) (*vault.Vault, *Distribution) {
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

	distribution, err := Open(v, now)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	return v, distribution
}

func TestPostingStoredSchedulesPosting(t *testing.T) {
	now := time.Unix(1000, 0)
	v, distribution := openDistribution(t, func() time.Time { return now })
	word, url := yacymodel.WordHash("w1"), urlHash("u1")

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return distribution.PostingStored(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingStored: %v", err)
	}

	due, err := distribution.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word]", due)
	}
}

func TestPostingPurgedFansOutToScheduleAndReplicas(t *testing.T) {
	now := time.Unix(1000, 0)
	v, distribution := openDistribution(t, func() time.Time { return now })
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return distribution.PostingStored(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingStored: %v", err)
	}
	if err := distribution.replicas.RecordAccepted(
		context.Background(), peer, []yacymodel.RWIPosting{{WordHash: word, URLHash: url}},
	); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return distribution.PostingPurged(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingPurged: %v", err)
	}

	due, err := distribution.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none after purge", due)
	}

	replicas, err := distribution.replicas.Holders(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Holders: %v", err)
	}
	if len(replicas) != 0 {
		t.Fatalf("replicas = %v, want none after purge", replicas)
	}
}

func TestCycleReturnsRunner(t *testing.T) {
	now := time.Unix(1000, 0)
	_, distribution := openDistribution(t, func() time.Time { return now })

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}

	runner := distribution.Cycle(
		http.DefaultClient,
		fakePostingIndex{},
		fakeRoster{},
		fakeURLDirectory{},
		distributioncycle.NoObserver{},
		Config{
			NetworkName:      "freeworld",
			Self:             yacymodel.WordHash("self"),
			Redundancy:       1,
			Partitions:       partitions,
			PostingsPerCycle: 10,
			CycleInterval:    time.Minute,
			RefreshInterval:  time.Hour,
			RetryInterval:    time.Minute,
		},
	)
	if runner == nil {
		t.Fatal("Cycle returned nil runner")
	}
}
