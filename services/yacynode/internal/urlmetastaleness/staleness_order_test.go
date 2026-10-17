package urlmetastaleness_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmetastaleness"
)

type harness struct {
	vault    *vault.Vault
	order    urlmetastaleness.StaleURLSource
	admitter urlmeta.URLMetadataAdmitter
	evictor  urlmeta.URLEvictor
}

func openOrder(t *testing.T) harness {
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

	order, projection, err := urlmetastaleness.Open(v)
	if err != nil {
		t.Fatalf("Open order: %v", err)
	}
	_, evictor, admitter, _, err := urlmeta.Open(
		context.Background(),
		v,
		time.Now,
		[]vault.Projection[yacymodel.URLMetadata]{projection},
	)
	if err != nil {
		t.Fatalf("urlmeta.Open: %v", err)
	}

	return harness{vault: v, order: order, admitter: admitter, evictor: evictor}
}

func (h harness) store(
	t *testing.T,
	hash yacymodel.URLHash,
	loaded yacymodel.Optional[yacymodel.CalendarDay],
) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		return h.admitter.Admit(tx, yacymodel.URLMetadata{Hash: hash, Loaded: loaded})
	})
}

func (h harness) write(t *testing.T, change func(tx *vault.Txn) error) {
	t.Helper()

	if err := h.vault.Update(context.Background(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (h harness) purge(t *testing.T, hash yacymodel.URLHash) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		_, err := h.evictor.Purge(context.Background(), tx, []yacymodel.URLHash{hash})

		return err
	})
}

func (h harness) stalestURLs(t *testing.T, limit int) []yacymodel.URLHash {
	t.Helper()

	var stalest []yacymodel.URLHash
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		selected, err := h.order.StalestURLs(tx, limit)
		stalest = selected

		return err
	}); err != nil {
		t.Fatalf("StalestURLs: %v", err)
	}

	return stalest
}

func dayIn(year int) yacymodel.Optional[yacymodel.CalendarDay] {
	return yacymodel.Some(yacymodel.NewCalendarDay(year, time.January, 1))
}

func TestStalestURLsReturnsStalestFirst(t *testing.T) {
	h := openOrder(t)
	fresh := urlHash("fresh")
	stale := urlHash("stale")
	middle := urlHash("middle")
	h.store(t, fresh, dayIn(2026))
	h.store(t, stale, dayIn(2020))
	h.store(t, middle, dayIn(2023))

	candidates := h.stalestURLs(t, 2)
	if len(candidates) != 2 || candidates[0] != stale || candidates[1] != middle {
		t.Fatalf("candidates = %v, want [stale middle]", candidates)
	}
}

func TestStalestURLsTreatsMissingFreshnessAsStalest(t *testing.T) {
	h := openOrder(t)
	dated := urlHash("dated")
	undated := urlHash("undated")
	h.store(t, dated, dayIn(2020))
	h.store(t, undated, yacymodel.None[yacymodel.CalendarDay]())

	candidates := h.stalestURLs(t, 1)
	if len(candidates) != 1 || candidates[0] != undated {
		t.Fatalf("candidates = %v, want [undated]", candidates)
	}
}

func TestStalestURLsZeroLimit(t *testing.T) {
	h := openOrder(t)

	candidates := h.stalestURLs(t, 0)
	if candidates != nil {
		t.Fatalf("candidates = %v, want nil", candidates)
	}
}

func TestPurgedURLLeavesOrder(t *testing.T) {
	h := openOrder(t)
	kept := urlHash("kept")
	gone := urlHash("gone")
	h.store(t, kept, dayIn(2025))
	h.store(t, gone, dayIn(2020))

	h.purge(t, gone)

	candidates := h.stalestURLs(t, 10)
	if len(candidates) != 1 || candidates[0] != kept {
		t.Fatalf("candidates = %v, want [kept]", candidates)
	}
}

func TestPurgeUnknownURLIsHarmless(t *testing.T) {
	h := openOrder(t)

	h.purge(t, urlHash("absent"))

	if candidates := h.stalestURLs(t, 10); len(candidates) != 0 {
		t.Fatalf("candidates = %v, want none", candidates)
	}
}

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}

func TestAURLStoredAgainOnAnotherDayKeepsOnlyItsNewPlace(t *testing.T) {
	h := openOrder(t)
	moved := urlHash("moved")
	kept := urlHash("kept")
	h.store(t, moved, dayIn(2020))
	h.store(t, kept, dayIn(2023))

	h.store(t, moved, dayIn(2026))

	candidates := h.stalestURLs(t, 10)
	if len(candidates) != 2 || candidates[0] != kept || candidates[1] != moved {
		t.Fatalf("candidates = %v, want [kept moved]", candidates)
	}
}
