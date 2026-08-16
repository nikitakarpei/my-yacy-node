package rwiescrow_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultengines/memory"
)

const (
	holdFor             = 5 * time.Minute
	roomForEveryPosting = 100
)

type countingHolds struct {
	held     int
	released int
}

func (c *countingHolds) ObserveHeld(postings int)     { c.held += postings }
func (c *countingHolds) ObserveReleased(postings int) { c.released += postings }

type harness struct {
	vault    *vault.Vault
	index    rwipostings.PostingIndex
	escrow   *rwiescrow.PostingEscrow
	observer *countingHolds
	clock    time.Time
}

func openHarness(t *testing.T) *harness {
	t.Helper()

	return openCappedHarness(t, roomForEveryPosting)
}

func openCappedHarness(t *testing.T, capacity int) *harness {
	t.Helper()

	v, err := memory.Open(0, nil)
	if err != nil {
		t.Fatalf("memory.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	index, admitter, _, err := rwipostings.Open(v)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}

	h := &harness{
		vault:    v,
		index:    index,
		observer: &countingHolds{},
		clock:    time.Unix(1700000000, 123456789),
	}
	escrow, err := rwiescrow.Open(
		v,
		admitter,
		h.observer,
		capacity,
		func() time.Time { return h.clock },
	)
	if err != nil {
		t.Fatalf("rwiescrow.Open: %v", err)
	}
	h.escrow = escrow

	return h
}

func (h *harness) hold(t *testing.T, postings ...yacymodel.RWIPosting) {
	t.Helper()

	if err := h.holding(postings...); err != nil {
		t.Fatalf("Hold: %v", err)
	}
}

func (h *harness) holding(postings ...yacymodel.RWIPosting) error {
	return h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		for _, entry := range postings {
			if err := h.escrow.Hold(tx, entry); err != nil {
				return fmt.Errorf("hold: %w", err)
			}
		}

		return nil
	})
}

func (h *harness) storeURL(t *testing.T, hash yacymodel.URLHash) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return h.escrow.URLStored(tx, hash, yacymodel.Optional[yacymodel.CalendarDay]{})
	}); err != nil {
		t.Fatalf("URLStored: %v", err)
	}
}

func (h *harness) indexed(t *testing.T, entry yacymodel.RWIPosting) bool {
	t.Helper()

	_, found, err := h.index.PostingOf(context.Background(), entry.WordHash, entry.URLHash)
	if err != nil {
		t.Fatalf("Posting: %v", err)
	}

	return found
}

func (h *harness) escrowedCount(t *testing.T) int {
	t.Helper()

	count, err := h.escrow.Count(context.Background())
	if err != nil {
		t.Fatalf("Count: %v", err)
	}

	return count
}

func urlHash(seed string) yacymodel.URLHash {
	address, err := url.Parse("http://example.com/" + seed)
	if err != nil {
		panic(err)
	}

	return yacymodel.URLNormalformOf(address).Hash()
}

func posting(word, urlSeed string) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash:   yacymodel.WordHash(word),
		URLHash:    urlHash(urlSeed),
		LocalLinks: 1,
		Hits:       1,
	}
}

func TestEscrowedPostingStaysOutOfIndex(t *testing.T) {
	h := openHarness(t)
	entry := posting("w1", "u1")

	h.hold(t, entry)

	if h.indexed(t, entry) {
		t.Fatal("escrowed posting reached the index before its url metadata arrived")
	}
	if got := h.escrowedCount(t); got != 1 {
		t.Fatalf("escrowed count = %d, want 1", got)
	}
	if h.observer.held != 1 {
		t.Fatalf("observed holds = %d, want 1", h.observer.held)
	}
}

func TestStoredURLReleasesEveryPostingWaitingForIt(t *testing.T) {
	h := openHarness(t)
	escrowed := []yacymodel.RWIPosting{
		posting("w1", "u1"),
		posting("w2", "u1"),
		posting("w1", "u2"),
	}
	waiting, other := escrowed[:2], escrowed[2]

	h.hold(t, escrowed...)
	h.storeURL(t, urlHash("u1"))

	for _, entry := range waiting {
		if !h.indexed(t, entry) {
			t.Fatalf(
				"posting for word %q not admitted after its url metadata arrived",
				entry.WordHash,
			)
		}
	}
	if h.indexed(t, other) {
		t.Fatal("posting for another url was admitted")
	}
	if got := h.escrowedCount(t); got != 1 {
		t.Fatalf("escrowed count = %d, want 1 posting still waiting", got)
	}
	if h.observer.released != 2 {
		t.Fatalf("observed releases = %d, want 2", h.observer.released)
	}
}

func TestEscrowedPostingExpiresAfterHoldPeriod(t *testing.T) {
	ctx := context.Background()
	h := openHarness(t)
	entry := posting("w1", "u1")

	h.hold(t, entry)

	expired, err := h.escrow.Expire(ctx, holdFor, 10)
	if err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if expired != 0 {
		t.Fatalf("expired = %d within the hold period, want 0", expired)
	}

	h.clock = h.clock.Add(holdFor + time.Second)

	expired, err = h.escrow.Expire(ctx, holdFor, 10)
	if err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if expired != 1 {
		t.Fatalf("expired = %d after the hold period, want 1", expired)
	}
	if h.indexed(t, entry) {
		t.Fatal("expired posting reached the index")
	}
	if got := h.escrowedCount(t); got != 0 {
		t.Fatalf("escrowed count = %d after expiry, want 0", got)
	}
}

func TestPostingHeldOutsideTheNanosecondEpochRangeExpiresOnTime(t *testing.T) {
	ctx := context.Background()
	h := openHarness(t)
	h.clock = time.Date(2500, time.January, 1, 0, 0, 0, 123456789, time.UTC)
	entry := posting("w1", "u1")

	h.hold(t, entry)
	h.clock = h.clock.Add(holdFor / 2)
	h.hold(t, entry)
	h.clock = h.clock.Add(holdFor/2 + time.Second)

	expired, err := h.escrow.Expire(ctx, holdFor, 10)
	if err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if expired != 0 {
		t.Fatalf(
			"expired = %d, want 0 while the refreshed hold is still within its period",
			expired,
		)
	}

	h.clock = h.clock.Add(holdFor)

	expired, err = h.escrow.Expire(ctx, holdFor, 10)
	if err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if expired != 1 {
		t.Fatalf("expired = %d after the hold period, want 1", expired)
	}
}

func TestReHoldRefreshesTheHoldInsteadOfDuplicatingIt(t *testing.T) {
	h := openHarness(t)
	entry := posting("w1", "u1")

	h.hold(t, entry)
	h.clock = h.clock.Add(holdFor / 2)
	h.hold(t, entry)

	if h.observer.held != 1 {
		t.Fatalf("observed holds = %d, want 1 for a re-hold of the same posting", h.observer.held)
	}
	if got := h.escrowedCount(t); got != 1 {
		t.Fatalf("escrowed count = %d, want 1", got)
	}

	h.clock = h.clock.Add(holdFor/2 + time.Second)
	expired, err := h.escrow.Expire(context.Background(), holdFor, 10)
	if err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if expired != 0 {
		t.Fatalf(
			"expired = %d at the original hold's deadline, want 0 since the hold was refreshed",
			expired,
		)
	}

	h.clock = h.clock.Add(holdFor)
	expired, err = h.escrow.Expire(context.Background(), holdFor, 10)
	if err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if expired != 1 {
		t.Fatalf("expired = %d after the refreshed hold's deadline, want 1", expired)
	}
}

const cappedCapacity = 3

func TestHoldReportsTheEscrowIsFullAtCapacity(t *testing.T) {
	h := openCappedHarness(t, cappedCapacity)
	if got := h.escrow.Capacity(); got != cappedCapacity {
		t.Fatalf("capacity = %d, want the configured %d", got, cappedCapacity)
	}
	admitted := postingsNumbering(cappedCapacity)
	h.hold(t, admitted...)

	if err := h.holding(posting("w1", "beyond")); !errors.Is(err, rwiescrow.ErrEscrowFull) {
		t.Fatalf("Hold at the capacity = %v, want ErrEscrowFull", err)
	}
	if got := h.escrowedCount(t); got != cappedCapacity {
		t.Fatalf("escrowed count = %d, want %d at the capacity", got, cappedCapacity)
	}

	h.clock = h.clock.Add(holdFor / 2)

	if err := h.holding(admitted[0]); err != nil {
		t.Fatalf("re-hold at the capacity = %v, want it to pass", err)
	}
}

func postingsNumbering(postings int) []yacymodel.RWIPosting {
	numbered := make([]yacymodel.RWIPosting, postings)
	for position := range numbered {
		numbered[position] = posting("w1", fmt.Sprintf("u%d", position))
	}

	return numbered
}

func TestExpireStopsAtLimit(t *testing.T) {
	h := openHarness(t)
	h.hold(t, posting("w1", "u1"), posting("w1", "u2"), posting("w1", "u3"))
	h.clock = h.clock.Add(holdFor + time.Second)

	expired, err := h.escrow.Expire(context.Background(), holdFor, 2)
	if err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if expired != 2 {
		t.Fatalf("expired = %d, want 2 at the limit", expired)
	}
	if got := h.escrowedCount(t); got != 1 {
		t.Fatalf("escrowed count = %d, want 1 left for the next run", got)
	}
}

func TestExpireTakesTheOldestHoldFirst(t *testing.T) {
	ctx := context.Background()
	h := openHarness(t)
	oldest, newest := posting("w1", "u1"), posting("w1", "u2")

	h.hold(t, oldest)
	h.clock = h.clock.Add(time.Minute)
	h.hold(t, newest)
	h.clock = h.clock.Add(holdFor + time.Second)

	expired, err := h.escrow.Expire(ctx, holdFor, 1)
	if err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if expired != 1 {
		t.Fatalf("expired = %d, want 1 at the limit", expired)
	}

	h.storeURL(t, oldest.URLHash)
	if h.indexed(t, oldest) {
		t.Fatal("the oldest hold outlived an expiry run that stopped at one posting")
	}

	h.storeURL(t, newest.URLHash)
	if !h.indexed(t, newest) {
		t.Fatal("the newer hold expired before the oldest one")
	}
}

func TestReleasedPostingIsNotExpired(t *testing.T) {
	h := openHarness(t)
	entry := posting("w1", "u1")

	h.hold(t, entry)
	h.storeURL(t, entry.URLHash)
	h.clock = h.clock.Add(holdFor + time.Second)

	expired, err := h.escrow.Expire(context.Background(), holdFor, 10)
	if err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if expired != 0 {
		t.Fatalf("expired = %d, want 0 after release dropped both rows", expired)
	}
	if !h.indexed(t, entry) {
		t.Fatal("released posting left the index")
	}
}

func TestPurgedURLLeavesEscrowedPostingsAlone(t *testing.T) {
	h := openHarness(t)
	entry := posting("w1", "u1")

	h.hold(t, entry)
	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return h.escrow.URLPurged(tx, entry.URLHash)
	}); err != nil {
		t.Fatalf("URLPurged: %v", err)
	}

	if got := h.escrowedCount(t); got != 1 {
		t.Fatalf("escrowed count = %d, want 1", got)
	}
}
