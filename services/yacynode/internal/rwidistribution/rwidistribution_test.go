package rwidistribution_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type postingRecordsHarness struct {
	vault    *vault.Vault
	schedule *postingofferschedule.Schedule
	replicas *postingreplicas.Replicas
	records  rwipostings.PostingObserver
}

func openPostingRecords(t *testing.T) postingRecordsHarness {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("memoryvault.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	schedule, replicas, records, err := rwidistribution.Open(
		v, frozenNow, discardedScheduleObservations{},
	)
	if err != nil {
		t.Fatalf("rwidistribution.Open: %v", err)
	}

	return postingRecordsHarness{
		vault:    v,
		schedule: schedule,
		replicas: replicas,
		records:  records,
	}
}

func (h postingRecordsHarness) update(t *testing.T, write func(tx *vault.Txn) error) {
	t.Helper()

	if err := h.vault.Update(context.Background(), write); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (h postingRecordsHarness) duePostings(t *testing.T) []postingidentity.Identity {
	t.Helper()

	var due []postingidentity.Identity
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		due, err = h.schedule.DuePostings(tx, 10)

		return err
	}); err != nil {
		t.Fatalf("DuePostings: %v", err)
	}

	return due
}

func (h postingRecordsHarness) holdersOf(
	t *testing.T,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) []yacymodel.Hash {
	t.Helper()

	var holders []yacymodel.Hash
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		holders, err = h.replicas.HoldersOf(tx, postingidentity.Identity{Word: word, URL: url})

		return err
	}); err != nil {
		t.Fatalf("HoldersOf: %v", err)
	}

	return holders
}

func frozenNow() time.Time { return time.Unix(1000, 0) }

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}

type discardedScheduleObservations struct{}

func (discardedScheduleObservations) ObserveScheduledPostings(string, int) {}

func (discardedScheduleObservations) ObserveLongestOfferLateness(string, time.Duration) {}

func TestPostingStoredSchedulesPosting(t *testing.T) {
	harness := openPostingRecords(t)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")

	harness.update(t, func(tx *vault.Txn) error {
		return harness.records.PostingStored(tx, yacymodel.RWIPosting{WordHash: word, URLHash: url})
	})

	due := harness.duePostings(t)
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want the stored posting %v", due, word)
	}
}

func TestAChangedPostingDropsTheReplicaLedgerAndIsDueAgain(t *testing.T) {
	harness := openPostingRecords(t)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	posting := yacymodel.RWIPosting{WordHash: word, URLHash: url}
	changedPosting := yacymodel.RWIPosting{WordHash: word, URLHash: url, Hits: 7}

	harness.update(t, func(tx *vault.Txn) error {
		return harness.records.PostingStored(tx, posting)
	})
	harness.update(t, func(tx *vault.Txn) error {
		return harness.replicas.RecordAccepted(tx, peer, []yacymodel.RWIPosting{posting})
	})

	harness.update(t, func(tx *vault.Txn) error {
		if err := harness.records.PostingPurged(tx, posting); err != nil {
			return err
		}

		return harness.records.PostingStored(tx, changedPosting)
	})

	due := harness.duePostings(t)
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want the changed posting %v", due, word)
	}
	if holders := harness.holdersOf(t, word, url); len(holders) != 0 {
		t.Fatalf("holders = %v, want none once the posting changed", holders)
	}
}

func TestPostingPurgedFansOutToScheduleAndReplicas(t *testing.T) {
	harness := openPostingRecords(t)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")

	harness.update(t, func(tx *vault.Txn) error {
		return harness.records.PostingStored(tx, yacymodel.RWIPosting{WordHash: word, URLHash: url})
	})
	harness.update(t, func(tx *vault.Txn) error {
		return harness.replicas.RecordAccepted(
			tx, peer, []yacymodel.RWIPosting{{WordHash: word, URLHash: url}},
		)
	})

	harness.update(t, func(tx *vault.Txn) error {
		return harness.records.PostingPurged(tx, yacymodel.RWIPosting{WordHash: word, URLHash: url})
	})

	if due := harness.duePostings(t); len(due) != 0 {
		t.Fatalf("due = %v, want none after purge", due)
	}
	if holders := harness.holdersOf(t, word, url); len(holders) != 0 {
		t.Fatalf("holders = %v, want none after purge", holders)
	}
}
