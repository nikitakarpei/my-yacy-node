package vault_test

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
	pagevisitrecordsvault "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecords/vault"
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

type silentExpiryObserver struct{}

func (silentExpiryObserver) ExpiryPassBegan(context.Context, vault.Name, time.Time) {}

func (silentExpiryObserver) ExpiryPassFinished(context.Context, vault.Name, time.Time) {}

func (silentExpiryObserver) ExpiredRecordsForgotten(context.Context, vault.Name, int) {}

func (silentExpiryObserver) ExpiredRecordsNotForgotten(context.Context, vault.Name, error) {}

type refusingEngine struct {
	vault.Engine
	refusing atomic.Bool
}

func (engine *refusingEngine) Update(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if engine.refusing.Load() {
		return errEngineRefusing
	}
	return engine.Engine.Update(ctx, fn)
}

func (engine *refusingEngine) View(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if engine.refusing.Load() {
		return errEngineRefusing
	}
	return engine.Engine.View(ctx, fn)
}

type recordedFailures struct{ causes []error }

func (f *recordedFailures) PageVisitNotRecorded(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	cause error,
) {
	f.causes = append(f.causes, cause)
}

func (f *recordedFailures) LastPageVisitNotRead(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	cause error,
) {
	f.causes = append(f.causes, cause)
}

type scene struct {
	records  *pagevisitrecordsvault.PageVisitRecords
	clock    *settableClock
	engine   *refusingEngine
	failures *recordedFailures
}

func pageVisitRecords(t *testing.T) scene {
	t.Helper()
	engine := &refusingEngine{Engine: memoryvault.OpenEngine(0)}
	clock := newSettableClock(visitedAt)
	storage, err := vault.New(engine, nil, clock)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	failures := &recordedFailures{}
	records, err := pagevisitrecordsvault.New(
		t.Context(),
		storage,
		clock,
		keptFor,
		failures,
		silentExpiryObserver{},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return scene{records: records, clock: clock, engine: engine, failures: failures}
}

func pageURL(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	return canonicalurltest.CanonicalURLOf(t, "http://example.com/a")
}

func TestNoPageVisitOfAPageNeverVisited(t *testing.T) {
	s := pageVisitRecords(t)

	if visit, visited := s.records.LastPageVisitOf(t.Context(), pageURL(t)); visited {
		t.Fatalf("want no page visit, got %+v", visit)
	}
}

func TestARecordedPageVisitKeepsItsTimeAndPageVersion(t *testing.T) {
	s := pageVisitRecords(t)
	version := pagefetch.PageVersion{EntityTag: `"abc"`, ModifiedAt: visitedAt.Add(-time.Minute)}
	s.records.RecordPageVisit(t.Context(), pageURL(t), version)

	s.clock.set(visitedAt.Add(30 * time.Minute))
	visit, visited := s.records.LastPageVisitOf(t.Context(), pageURL(t))

	if !visited || !visit.VisitedAt.Equal(visitedAt) {
		t.Fatalf("visited at %v, want the page visit recorded at %v", visit.VisitedAt, visitedAt)
	}
	if visit.Version.EntityTag != version.EntityTag ||
		!visit.Version.ModifiedAt.Equal(version.ModifiedAt) {
		t.Fatalf("page version = %+v, want %+v", visit.Version, version)
	}
}

func TestALastPageVisitTheStorageRefusesIsReportedAsNeverVisited(t *testing.T) {
	s := pageVisitRecords(t)
	s.records.RecordPageVisit(t.Context(), pageURL(t), pagefetch.PageVersion{})
	s.engine.refusing.Store(true)

	if visit, visited := s.records.LastPageVisitOf(t.Context(), pageURL(t)); visited {
		t.Fatalf("want no page visit, got %+v", visit)
	}
	if len(s.failures.causes) != 1 || !errors.Is(s.failures.causes[0], errEngineRefusing) {
		t.Fatalf("reported %v, want the refusal of the storage", s.failures.causes)
	}
}

func TestAPageVisitTheStorageRefusesIsReported(t *testing.T) {
	s := pageVisitRecords(t)
	s.engine.refusing.Store(true)

	s.records.RecordPageVisit(t.Context(), pageURL(t), pagefetch.PageVersion{})

	if len(s.failures.causes) != 1 || !errors.Is(s.failures.causes[0], errEngineRefusing) {
		t.Fatalf("reported %v, want the refusal of the storage", s.failures.causes)
	}
}

func TestAPageVisitIsForgottenOnceItsRetentionPasses(t *testing.T) {
	s := pageVisitRecords(t)
	s.records.RecordPageVisit(t.Context(), pageURL(t), pagefetch.PageVersion{})

	s.clock.set(visitedAt.Add(keptFor))

	if visit, visited := s.records.LastPageVisitOf(t.Context(), pageURL(t)); visited {
		t.Fatalf("a page visit past its retention read back as %+v", visit)
	}
}
