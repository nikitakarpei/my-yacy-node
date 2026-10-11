package postingofferschedule_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferinterval"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
)

var testInterval = postingofferinterval.Bounds{
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

const (
	shortfall = string(postingofferschedule.OfferOrderShortfall)
	refresh   = string(postingofferschedule.OfferOrderRefresh)
)

type recordedObservations struct {
	scheduled map[string]int
	lateness  map[string]time.Duration
}

func (o *recordedObservations) ObserveScheduledPostings(order string, postings int) {
	o.scheduled[order] = postings
}

func (o *recordedObservations) ObserveLongestOfferLateness(order string, lateness time.Duration) {
	o.lateness[order] = lateness
}

type scheduleHarness struct {
	engine   vault.Engine
	vault    *vault.Vault
	schedule *postingofferschedule.Schedule
	observed *recordedObservations
	clock    time.Time
}

func openSchedule(t *testing.T, clockStart time.Time) *scheduleHarness {
	t.Helper()

	engine := memoryvault.OpenEngine(0)
	t.Cleanup(func() {
		if err := engine.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	harness := &scheduleHarness{engine: engine, observed: &recordedObservations{
		scheduled: map[string]int{},
		lateness:  map[string]time.Duration{},
	}, clock: clockStart}
	harness.reopenWith(t, yacymodel.DHTRingPartitions(1))

	return harness
}

func (h *scheduleHarness) reopenWith(t *testing.T, partitions yacymodel.DHTRingPartitions) {
	t.Helper()

	v, err := vault.New(h.engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	schedule, err := postingofferschedule.Open(
		v,
		partitions,
		testInterval,
		func() time.Time { return h.clock },
		h.observed,
	)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	h.vault, h.schedule = v, schedule
}

func wordIn(sector yacymodel.DHTRingSector) yacymodel.Hash {
	return wordsIn(sector, 1)[0]
}

func wordsIn(sector yacymodel.DHTRingSector, count int) []yacymodel.Hash {
	var words []yacymodel.Hash
	for attempt := 0; len(words) < count; attempt++ {
		word := yacymodel.WordHash(fmt.Sprintf("word-%d", attempt))
		if yacymodel.DHTRingSectorOf(yacymodel.DHTRingPositionOf(word)) == sector {
			words = append(words, word)
		}
	}

	return words
}

func (h *scheduleHarness) store(t *testing.T, word yacymodel.Hash, url yacymodel.URLHash) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return h.schedule.PostingStored(tx, yacymodel.RWIPosting{WordHash: word, URLHash: url})
	}); err != nil {
		t.Fatalf("PostingStored: %v", err)
	}
}

func (h *scheduleHarness) purge(t *testing.T, word yacymodel.Hash, url yacymodel.URLHash) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return h.schedule.PostingPurged(tx, yacymodel.RWIPosting{WordHash: word, URLHash: url})
	}); err != nil {
		t.Fatalf("PostingPurged: %v", err)
	}
}

func (h *scheduleHarness) pauseOffer(
	t *testing.T,
	word yacymodel.Hash,
	url yacymodel.URLHash,
	requestedPause time.Duration,
) bool {
	t.Helper()

	var wasRescheduled bool
	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		var err error
		wasRescheduled, err = h.schedule.SetNextOfferAfterRedundancyMissed(
			tx,
			postingidentity.Identity{Word: word, URL: url},
			requestedPause,
		)

		return err
	}); err != nil {
		t.Fatalf("SetNextOfferAfterRedundancyMissed: %v", err)
	}

	return wasRescheduled
}

func (h *scheduleHarness) meetRedundancy(
	t *testing.T,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) bool {
	t.Helper()

	var wasRescheduled bool
	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		var err error
		wasRescheduled, err = h.schedule.SetNextOfferAfterRedundancyMet(
			tx,
			postingidentity.Identity{Word: word, URL: url},
		)

		return err
	}); err != nil {
		t.Fatalf("SetNextOfferAfterRedundancyMet: %v", err)
	}

	return wasRescheduled
}

func (h *scheduleHarness) observeBacklog(t *testing.T) {
	t.Helper()

	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		return h.schedule.ObserveBacklog(tx)
	}); err != nil {
		t.Fatalf("ObserveBacklog: %v", err)
	}
}

func (h *scheduleHarness) duePostings(t *testing.T, limit int) []postingidentity.Identity {
	t.Helper()

	var due []postingidentity.Identity
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		for sector := range yacymodel.DHTRingSectorCount {
			inSector, err := h.schedule.DuePostingsIn(tx, sector, limit-len(due))
			if err != nil {
				return err
			}
			due = append(due, inSector...)
		}

		return nil
	}); err != nil {
		t.Fatalf("due postings: %v", err)
	}

	return due
}

func (h *scheduleHarness) duePostingsIn(
	t *testing.T,
	sector yacymodel.DHTRingSector,
	limit int,
) []postingidentity.Identity {
	t.Helper()

	var due []postingidentity.Identity
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		due, err = h.schedule.DuePostingsIn(tx, sector, limit)

		return err
	}); err != nil {
		t.Fatalf("DuePostingsIn: %v", err)
	}

	return due
}
