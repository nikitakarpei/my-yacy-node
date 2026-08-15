package postingofferschedule_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultengines/memory"
)

var testInterval = postingofferschedule.OfferInterval{
	Shortest: time.Minute,
	Longest:  8 * time.Minute,
}

var testStart = time.Unix(1_000_000, 0).UTC()

var testWord = yacymodel.WordHash("w1")

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
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

type postingOffers struct {
	vault    *vault.Vault
	schedule *postingofferschedule.Schedule
	observed *recordedObservations
	clock    time.Time
}

func openOffers(t *testing.T, clockStart time.Time) *postingOffers {
	t.Helper()

	v, err := memory.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	offers := &postingOffers{vault: v, observed: &recordedObservations{}, clock: clockStart}
	offers.schedule, err = postingofferschedule.Open(
		v,
		func() time.Time { return offers.clock },
		offers.observed,
	)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	return offers
}

func (o *postingOffers) store(t *testing.T, word yacymodel.Hash, url yacymodel.URLHash) {
	t.Helper()

	if err := o.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return o.schedule.PostingStored(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingStored: %v", err)
	}
}

func (o *postingOffers) purge(t *testing.T, word yacymodel.Hash, url yacymodel.URLHash) {
	t.Helper()

	if err := o.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return o.schedule.PostingPurged(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingPurged: %v", err)
	}
}

func (o *postingOffers) pauseOffer(
	t *testing.T,
	word yacymodel.Hash,
	url yacymodel.URLHash,
	requestedPause time.Duration,
) {
	t.Helper()

	if err := o.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return o.schedule.SetNextOfferAfterRedundancyMissed(
			tx,
			postingidentity.IdentityOf(word, url),
			testInterval,
			requestedPause,
		)
	}); err != nil {
		t.Fatalf("SetNextOfferAfterRedundancyMissed: %v", err)
	}
}

func (o *postingOffers) meetRedundancy(t *testing.T, word yacymodel.Hash, url yacymodel.URLHash) {
	t.Helper()

	if err := o.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return o.schedule.SetNextOfferAfterRedundancyMet(
			tx,
			postingidentity.IdentityOf(word, url),
			testInterval,
		)
	}); err != nil {
		t.Fatalf("SetNextOfferAfterRedundancyMet: %v", err)
	}
}

func (o *postingOffers) isScheduled(
	t *testing.T,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) bool {
	t.Helper()

	var postingScheduled bool
	if err := o.vault.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		postingScheduled, err = o.schedule.IsScheduled(tx, postingidentity.IdentityOf(word, url))

		return err
	}); err != nil {
		t.Fatalf("IsScheduled: %v", err)
	}

	return postingScheduled
}

func (o *postingOffers) duePostings(t *testing.T, limit int) []postingidentity.Identity {
	t.Helper()

	due, err := o.schedule.DuePostings(context.Background(), limit)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}

	return due
}

func (o *postingOffers) duePostingsAt(t *testing.T, at time.Time) int {
	t.Helper()

	o.clock = at

	return len(o.duePostings(t, 10))
}
