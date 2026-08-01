package rwidistribution

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

func openSchedule(t *testing.T, now func() time.Time) (*vault.Vault, *postingOfferSchedule) {
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

	return v, schedule
}

func store(
	t *testing.T,
	schedule *postingOfferSchedule,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) {
	t.Helper()

	if err := schedule.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return schedule.PostingStored(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingStored: %v", err)
	}
}

func TestPostingStoredIsImmediatelyDue(t *testing.T) {
	now := time.Unix(1000, 0)
	_, schedule := openSchedule(t, func() time.Time { return now })
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	store(t, schedule, word, url)

	due, err := schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 1 || due[0].Word != word || due[0].URL != url {
		t.Fatalf("due = %v, want single entry for %v/%v", due, word, url)
	}
}

func TestDuePostingsExcludesFutureEntries(t *testing.T) {
	now := time.Unix(1000, 0)
	_, schedule := openSchedule(t, func() time.Time { return now })
	overdue, future := yacymodel.WordHash("overdue"), yacymodel.WordHash("future")
	url := urlHash("u1")

	if err := schedule.Reschedule(
		context.Background(),
		overdue,
		url,
		now.Add(-time.Minute),
	); err != nil {
		t.Fatalf("Reschedule overdue: %v", err)
	}
	if err := schedule.Reschedule(
		context.Background(),
		future,
		url,
		now.Add(time.Hour),
	); err != nil {
		t.Fatalf("Reschedule future: %v", err)
	}

	due, err := schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 1 || due[0].Word != overdue {
		t.Fatalf("due = %v, want only [overdue]", due)
	}
}

func TestDuePostingsRespectsLimit(t *testing.T) {
	now := time.Unix(1000, 0)
	_, schedule := openSchedule(t, func() time.Time { return now })
	for _, seed := range []string{"a", "b", "c"} {
		if err := schedule.Reschedule(
			context.Background(),
			yacymodel.WordHash(seed),
			urlHash(seed),
			now.Add(-time.Minute),
		); err != nil {
			t.Fatalf("Reschedule %s: %v", seed, err)
		}
	}

	due, err := schedule.DuePostings(context.Background(), 2)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 2 {
		t.Fatalf("due = %v, want 2 entries", due)
	}
}

func TestRescheduleMovesOrderPosition(t *testing.T) {
	now := time.Unix(1000, 0)
	_, schedule := openSchedule(t, func() time.Time { return now })
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	store(t, schedule, word, url)

	if err := schedule.Reschedule(context.Background(), word, url, now.Add(time.Hour)); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}

	due, err := schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due after rescheduling into the future", due)
	}
}

func TestPostingPurgedRemovesBothRows(t *testing.T) {
	now := time.Unix(1000, 0)
	v, schedule := openSchedule(t, func() time.Time { return now })
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	store(t, schedule, word, url)

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return schedule.PostingPurged(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingPurged: %v", err)
	}

	due, err := schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none after purge", due)
	}

	var found bool
	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		_, ok, err := schedule.due.Get(tx, postingKey(word, url))
		found = ok

		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
	if found {
		t.Fatal("due row still present after purge")
	}
}

func TestForgetRemovesBothRows(t *testing.T) {
	now := time.Unix(1000, 0)
	_, schedule := openSchedule(t, func() time.Time { return now })
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	store(t, schedule, word, url)

	if err := schedule.Forget(context.Background(), word, url); err != nil {
		t.Fatalf("Forget: %v", err)
	}

	due, err := schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none after Forget", due)
	}
}

func TestForgetUnknownIsHarmless(t *testing.T) {
	_, schedule := openSchedule(t, time.Now)

	if err := schedule.Forget(
		context.Background(),
		yacymodel.WordHash("absent"),
		urlHash("absent"),
	); err != nil {
		t.Fatalf("Forget: %v", err)
	}
}

func TestPostingPurgedUnknownIsHarmless(t *testing.T) {
	_, schedule := openSchedule(t, time.Now)

	if err := schedule.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return schedule.PostingPurged(
			tx,
			yacymodel.WordHash("absent"),
			urlHash("absent"),
		)
	}); err != nil {
		t.Fatalf("PostingPurged: %v", err)
	}
}
