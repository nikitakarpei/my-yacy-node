package postingofferschedule

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

func openSchedule(
	t *testing.T,
	now func() time.Time,
) (*vault.Vault, *Schedule, *recordedObservations) {
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

	observed := &recordedObservations{}
	schedule, err := Open(v, now, observed)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	return v, schedule, observed
}

func store(
	t *testing.T,
	schedule *Schedule,
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

func rescheduleOffer(
	t *testing.T,
	schedule *Schedule,
	word yacymodel.Hash,
	url yacymodel.URLHash,
	at time.Time,
) {
	t.Helper()

	if err := schedule.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return schedule.reschedule(
			tx,
			postingidentity.IdentityOf(word, url),
			func(time.Time) time.Time { return at },
		)
	}); err != nil {
		t.Fatalf("reschedule: %v", err)
	}
}

func TestPostingStoredIsImmediatelyDue(t *testing.T) {
	now := time.Unix(1000, 0)
	_, schedule, _ := openSchedule(t, func() time.Time { return now })
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
	_, schedule, _ := openSchedule(t, func() time.Time { return now })
	overdue, future := yacymodel.WordHash("overdue"), yacymodel.WordHash("future")
	url := urlHash("u1")
	store(t, schedule, overdue, url)
	store(t, schedule, future, url)

	rescheduleOffer(t, schedule, overdue, url, now.Add(-time.Minute))
	rescheduleOffer(t, schedule, future, url, now.Add(time.Hour))

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
	_, schedule, _ := openSchedule(t, func() time.Time { return now })
	for _, seed := range []string{"a", "b", "c"} {
		word, url := yacymodel.WordHash(seed), urlHash(seed)
		store(t, schedule, word, url)
		rescheduleOffer(t, schedule, word, url, now.Add(-time.Minute))
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
	_, schedule, _ := openSchedule(t, func() time.Time { return now })
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	store(t, schedule, word, url)

	rescheduleOffer(t, schedule, word, url, now.Add(time.Hour))

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
	v, schedule, _ := openSchedule(t, func() time.Time { return now })
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
		_, ok, err := schedule.dueTimes.Get(tx, postingidentity.IdentityOf(word, url))
		found = ok

		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
	if found {
		t.Fatal("due row still present after purge")
	}
}

func TestRescheduleUnknownIsHarmless(t *testing.T) {
	_, schedule, _ := openSchedule(t, time.Now)

	rescheduleOffer(t, schedule, yacymodel.WordHash("absent"), urlHash("absent"), time.Now())
}

func TestRescheduleDoesNotResurrectPurgedPosting(t *testing.T) {
	now := time.Unix(1000, 0)
	v, schedule, _ := openSchedule(t, func() time.Time { return now })
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	store(t, schedule, word, url)

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return schedule.PostingPurged(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingPurged: %v", err)
	}

	rescheduleOffer(t, schedule, word, url, now.Add(time.Hour))

	due, err := schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none after purge and reschedule", due)
	}
}

func TestPostingPurgedUnknownIsHarmless(t *testing.T) {
	_, schedule, _ := openSchedule(t, time.Now)

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

type recordedObservations struct {
	scheduled int
	lateness  time.Duration
}

func (o *recordedObservations) ObserveScheduledPostings(postings int) {
	o.scheduled = postings
}

func (o *recordedObservations) ObserveLongestOfferLateness(lateness time.Duration) {
	o.lateness = lateness
}

func TestObserveCountsScheduledPostings(t *testing.T) {
	now := time.Unix(1000, 0)
	_, schedule, observed := openSchedule(t, func() time.Time { return now })
	word, url := yacymodel.WordHash("w1"), urlHash("u1")

	schedule.ObserveBacklog(context.Background())
	if observed.scheduled != 0 {
		t.Fatalf("scheduled = %d, want 0 for an empty schedule", observed.scheduled)
	}

	store(t, schedule, word, url)

	schedule.ObserveBacklog(context.Background())
	if observed.scheduled != 1 {
		t.Fatalf("scheduled = %d, want 1 after a posting is stored", observed.scheduled)
	}
}

func TestObserveReportsNoLatenessForEmptySchedule(t *testing.T) {
	_, schedule, observed := openSchedule(t, time.Now)

	schedule.ObserveBacklog(context.Background())

	if observed.lateness != 0 {
		t.Fatalf("lateness = %v, want 0 for an empty schedule", observed.lateness)
	}
}

func TestObserveReportsLatenessOfEarliestEntry(t *testing.T) {
	now := time.Unix(1000, 0)
	_, schedule, observed := openSchedule(t, func() time.Time { return now })
	earlier, url := yacymodel.WordHash("earlier"), urlHash("u1")
	later := yacymodel.WordHash("later")
	store(t, schedule, later, url)
	rescheduleOffer(t, schedule, later, url, now.Add(time.Hour))
	store(t, schedule, earlier, url)
	rescheduleOffer(t, schedule, earlier, url, now.Add(-time.Hour))

	schedule.ObserveBacklog(context.Background())

	if observed.lateness != time.Hour {
		t.Fatalf("lateness = %v, want %v", observed.lateness, time.Hour)
	}
}
