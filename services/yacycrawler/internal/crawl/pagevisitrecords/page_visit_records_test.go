package pagevisitrecords_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitrecords"
)

const keptFor = time.Hour

var (
	visitedAt         = time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	errEngineRefusing = errors.New("engine refuses")
)

type settableClock struct {
	now atomic.Pointer[time.Time]
}

func newSettableClock(now time.Time) *settableClock {
	clock := &settableClock{}
	clock.set(now)
	return clock
}

func (clock *settableClock) Now() time.Time { return *clock.now.Load() }

func (clock *settableClock) set(now time.Time) { clock.now.Store(&now) }

func (*settableClock) Sleep(context.Context, time.Duration) error { return context.Canceled }

type refusingEngine struct {
	vault.Engine
	refusing atomic.Bool
}

func (engine *refusingEngine) Update(ctx context.Context, fn func(vault.EngineTxn) error) error {
	return engine.Engine.Update(ctx, func(etx vault.EngineTxn) error {
		return fn(engine.txnOf(etx))
	})
}

func (engine *refusingEngine) View(ctx context.Context, fn func(vault.EngineTxn) error) error {
	return engine.Engine.View(ctx, func(etx vault.EngineTxn) error {
		return fn(engine.txnOf(etx))
	})
}

func (engine *refusingEngine) txnOf(etx vault.EngineTxn) vault.EngineTxn {
	if engine.refusing.Load() {
		return refusingTxn{EngineTxn: etx}
	}
	return etx
}

type refusingTxn struct {
	vault.EngineTxn
}

func (txn refusingTxn) ExpiringRecords(name vault.Name) vault.EngineExpiringRecords {
	return refusingExpiringRecords{EngineExpiringRecords: txn.EngineTxn.ExpiringRecords(name)}
}

type refusingExpiringRecords struct {
	vault.EngineExpiringRecords
}

func (refusingExpiringRecords) Get([]byte, time.Time) ([]byte, error) {
	return nil, errEngineRefusing
}

func (refusingExpiringRecords) Put([]byte, []byte, time.Time) ([]byte, error) {
	return nil, errEngineRefusing
}

type pageVisitRecordsPort interface {
	LastPageVisitOf(
		tx *vault.Txn,
		canonicalURL canonicalurl.CanonicalURL,
	) (pagevisit.LastPageVisit, bool, error)
	RecordPageVisit(
		tx *vault.Txn,
		canonicalURL canonicalurl.CanonicalURL,
		version pagefetch.PageVersion,
	) error
}

type scene struct {
	storage *vault.Vault
	records pageVisitRecordsPort
	clock   *settableClock
	engine  *refusingEngine
}

func pageVisitRecords(t *testing.T) scene {
	t.Helper()
	engine := &refusingEngine{Engine: memoryvault.OpenEngine(0)}
	clock := newSettableClock(visitedAt)
	storage, err := vault.New(engine, nil, nil, clock)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	records, err := pagevisitrecords.Open(t.Context(), storage, clock, keptFor)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return scene{storage: storage, records: records, clock: clock, engine: engine}
}

func pageURL(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	return canonicalurltest.CanonicalURLOf(t, "http://example.com/a")
}

func (s scene) lastPageVisit(t *testing.T) (pagevisit.LastPageVisit, bool, error) {
	t.Helper()
	var visit pagevisit.LastPageVisit
	var visited bool
	err := s.storage.View(t.Context(), func(tx *vault.Txn) error {
		var err error
		visit, visited, err = s.records.LastPageVisitOf(tx, pageURL(t))
		return err
	})
	return visit, visited, err
}

func (s scene) record(t *testing.T, version pagefetch.PageVersion) error {
	t.Helper()
	return s.storage.Update(t.Context(), func(tx *vault.Txn) error {
		return s.records.RecordPageVisit(tx, pageURL(t), version)
	})
}

func TestNoPageVisitOfAPageNeverVisited(t *testing.T) {
	s := pageVisitRecords(t)

	if visit, visited, err := s.lastPageVisit(t); err != nil || visited {
		t.Fatalf("want no page visit, got %+v (%v)", visit, err)
	}
}

func TestARecordedPageVisitKeepsItsTimeAndPageVersion(t *testing.T) {
	s := pageVisitRecords(t)
	version := pagefetch.PageVersion{EntityTag: `"abc"`, ModifiedAt: visitedAt.Add(-time.Minute)}
	if err := s.record(t, version); err != nil {
		t.Fatalf("RecordPageVisit: %v", err)
	}

	s.clock.set(visitedAt.Add(30 * time.Minute))
	visit, visited, err := s.lastPageVisit(t)

	if err != nil || !visited || !visit.VisitedAt.Equal(visitedAt) {
		t.Fatalf(
			"visited at %v (%v), want the page visit recorded at %v",
			visit.VisitedAt,
			err,
			visitedAt,
		)
	}
	if visit.Version.EntityTag != version.EntityTag ||
		!visit.Version.ModifiedAt.Equal(version.ModifiedAt) {
		t.Fatalf("page version = %+v, want %+v", visit.Version, version)
	}
}

func TestALastPageVisitTheStorageRefusesFails(t *testing.T) {
	s := pageVisitRecords(t)
	if err := s.record(t, pagefetch.PageVersion{}); err != nil {
		t.Fatalf("RecordPageVisit: %v", err)
	}
	s.engine.refusing.Store(true)

	if _, _, err := s.lastPageVisit(t); !errors.Is(err, errEngineRefusing) {
		t.Fatalf("LastPageVisitOf err = %v, want the refusal of the storage", err)
	}
}

func TestAPageVisitTheStorageRefusesFails(t *testing.T) {
	s := pageVisitRecords(t)
	s.engine.refusing.Store(true)

	if err := s.record(t, pagefetch.PageVersion{}); !errors.Is(err, errEngineRefusing) {
		t.Fatalf("RecordPageVisit err = %v, want the refusal of the storage", err)
	}
}

func TestAPageVisitIsForgottenOnceItsRetentionPasses(t *testing.T) {
	s := pageVisitRecords(t)
	if err := s.record(t, pagefetch.PageVersion{}); err != nil {
		t.Fatalf("RecordPageVisit: %v", err)
	}

	s.clock.set(visitedAt.Add(keptFor))

	if visit, visited, err := s.lastPageVisit(t); err != nil || visited {
		t.Fatalf("a page visit past its retention read back as %+v (%v)", visit, err)
	}
}
