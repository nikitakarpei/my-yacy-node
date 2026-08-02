package rwidistribution

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}

func openRecords(t *testing.T, now func() time.Time) (*vault.Vault, *postingRecords) {
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

	schedule, replicas, _, err := Open(v, now, discardedScheduleObservations{})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	return v, &postingRecords{schedule: schedule, replicas: replicas}
}

func TestPostingStoredSchedulesPosting(t *testing.T) {
	now := time.Unix(1000, 0)
	v, records := openRecords(t, func() time.Time { return now })
	word, url := yacymodel.WordHash("w1"), urlHash("u1")

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return records.PostingStored(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingStored: %v", err)
	}

	due, err := records.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word]", due)
	}
}

func TestPostingPurgedFansOutToScheduleAndReplicas(t *testing.T) {
	now := time.Unix(1000, 0)
	v, records := openRecords(t, func() time.Time { return now })
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return records.PostingStored(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingStored: %v", err)
	}
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return records.replicas.RecordAccepted(
			tx, peer, []yacymodel.RWIPosting{{WordHash: word, URLHash: url}},
		)
	}); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return records.PostingPurged(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingPurged: %v", err)
	}

	due, err := records.schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none after purge", due)
	}

	var replicas []yacymodel.Hash
	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		replicas, err = records.replicas.HoldersOf(tx, postingidentity.IdentityOf(word, url))

		return err
	}); err != nil {
		t.Fatalf("Holders: %v", err)
	}
	if len(replicas) != 0 {
		t.Fatalf("replicas = %v, want none after purge", replicas)
	}
}

type discardedScheduleObservations struct{}

func (discardedScheduleObservations) ObserveScheduledPostings(int) {}

func (discardedScheduleObservations) ObserveLongestOfferLateness(time.Duration) {}
