package rwidistribution

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

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

func TestPostingPurgedFansOutToScheduleAndLedger(t *testing.T) {
	now := time.Unix(1000, 0)
	v, distribution := openDistribution(t, func() time.Time { return now })
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return distribution.PostingStored(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingStored: %v", err)
	}
	if err := distribution.ledger.RecordAccepted(
		context.Background(),
		postingOffer{Peer: seed(peer), Postings: []yacymodel.RWIPosting{fakePosting(word, url)}},
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

	replicas, err := distribution.ledger.Replicas(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
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
		noPostingOfferCycleObserver{},
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
