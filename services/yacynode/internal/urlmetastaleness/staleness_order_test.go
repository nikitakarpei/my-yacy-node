package urlmetastaleness_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmetastaleness"
)

func openOrder(t *testing.T) (*vault.Vault, urlmetastaleness.StalenessRanking) {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	order, err := urlmetastaleness.Open(v)
	if err != nil {
		t.Fatalf("Open order: %v", err)
	}

	return v, order
}

func store(
	t *testing.T,
	v *vault.Vault,
	order urlmetastaleness.StalenessRanking,
	hash yacymodel.URLHash,
	freshness yacymodel.Optional[yacymodel.CalendarDay],
) {
	t.Helper()

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return order.URLStored(tx, hash, freshness)
	}); err != nil {
		t.Fatalf("URLStored: %v", err)
	}
}

func TestStalestURLsReturnsStalestFirst(t *testing.T) {
	vault, order := openOrder(t)
	fresh := urlHash("fresh")
	stale := urlHash("stale")
	middle := urlHash("middle")
	store(t, vault, order, fresh, yacymodel.Some(yacymodel.NewCalendarDay(2026, time.January, 1)))
	store(t, vault, order, stale, yacymodel.Some(yacymodel.NewCalendarDay(2020, time.January, 1)))
	store(t, vault, order, middle, yacymodel.Some(yacymodel.NewCalendarDay(2023, time.January, 1)))

	candidates, err := order.StalestURLs(context.Background(), 2)
	if err != nil {
		t.Fatalf("StalestURLs: %v", err)
	}
	if len(candidates) != 2 || candidates[0] != stale || candidates[1] != middle {
		t.Fatalf("candidates = %v, want [stale middle]", candidates)
	}
}

func TestStalestURLsTreatsMissingFreshnessAsStalest(t *testing.T) {
	vault, order := openOrder(t)
	dated := urlHash("dated")
	undated := urlHash("undated")
	store(t, vault, order, dated, yacymodel.Some(yacymodel.NewCalendarDay(2020, time.January, 1)))
	store(t, vault, order, undated, yacymodel.None[yacymodel.CalendarDay]())

	candidates, err := order.StalestURLs(context.Background(), 1)
	if err != nil {
		t.Fatalf("StalestURLs: %v", err)
	}
	if len(candidates) != 1 || candidates[0] != undated {
		t.Fatalf("candidates = %v, want [undated]", candidates)
	}
}

func TestStalestURLsZeroLimit(t *testing.T) {
	_, order := openOrder(t)

	candidates, err := order.StalestURLs(context.Background(), 0)
	if err != nil {
		t.Fatalf("StalestURLs: %v", err)
	}
	if candidates != nil {
		t.Fatalf("candidates = %v, want nil", candidates)
	}
}

func TestPurgedURLLeavesOrder(t *testing.T) {
	v, order := openOrder(t)
	kept := urlHash("kept")
	gone := urlHash("gone")
	store(t, v, order, kept, yacymodel.Some(yacymodel.NewCalendarDay(2025, time.January, 1)))
	store(t, v, order, gone, yacymodel.Some(yacymodel.NewCalendarDay(2020, time.January, 1)))

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return order.URLPurged(tx, gone)
	}); err != nil {
		t.Fatalf("URLPurged: %v", err)
	}

	candidates, err := order.StalestURLs(context.Background(), 10)
	if err != nil {
		t.Fatalf("StalestURLs: %v", err)
	}
	if len(candidates) != 1 || candidates[0] != kept {
		t.Fatalf("candidates = %v, want [kept]", candidates)
	}
}

func TestPurgeUnknownURLIsHarmless(t *testing.T) {
	v, order := openOrder(t)

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return order.URLPurged(tx, urlHash("absent"))
	}); err != nil {
		t.Fatalf("URLPurged: %v", err)
	}
}

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}
