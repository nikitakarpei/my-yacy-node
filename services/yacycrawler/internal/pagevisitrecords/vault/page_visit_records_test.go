package vault_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	pagevisitrecordsvault "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecords/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/retainedrecords"
)

const keptFor = time.Hour

var (
	visitedAt         = time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	errEngineRefusing = errors.New("engine refuses")
)

type settableClock struct {
	now time.Time
}

func (clock *settableClock) Now() time.Time { return clock.now }

func (*settableClock) Sleep(context.Context, time.Duration) error { return context.Canceled }

type silentExpiryObserver struct{}

func (silentExpiryObserver) ExpiredRecordsForgotten(context.Context, string, int) {}

func (silentExpiryObserver) ExpiredRecordsNotForgotten(context.Context, string, error) {}

type refusingEngine struct {
	vault.Engine
	refusing bool
}

func (engine *refusingEngine) Update(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if engine.refusing {
		return errEngineRefusing
	}
	return engine.Engine.Update(ctx, fn)
}

func (engine *refusingEngine) View(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if engine.refusing {
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
	storage, err := vault.New(engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	clock := &settableClock{now: visitedAt}
	failures := &recordedFailures{}
	records, err := pagevisitrecordsvault.New(
		t.Context(),
		storage,
		retainedrecords.New(storage, clock, silentExpiryObserver{}),
		clock,
		keptFor,
		failures,
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

	s.clock.now = visitedAt.Add(30 * time.Minute)
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
	s.engine.refusing = true

	if visit, visited := s.records.LastPageVisitOf(t.Context(), pageURL(t)); visited {
		t.Fatalf("want no page visit, got %+v", visit)
	}
	if len(s.failures.causes) != 1 || !errors.Is(s.failures.causes[0], errEngineRefusing) {
		t.Fatalf("reported %v, want the refusal of the storage", s.failures.causes)
	}
}

func TestAPageVisitTheStorageRefusesIsReported(t *testing.T) {
	s := pageVisitRecords(t)
	s.engine.refusing = true

	s.records.RecordPageVisit(t.Context(), pageURL(t), pagefetch.PageVersion{})

	if len(s.failures.causes) != 1 || !errors.Is(s.failures.causes[0], errEngineRefusing) {
		t.Fatalf("reported %v, want the refusal of the storage", s.failures.causes)
	}
}

func TestAPageVisitIsForgottenOnceItsRetentionPasses(t *testing.T) {
	s := pageVisitRecords(t)
	s.records.RecordPageVisit(t.Context(), pageURL(t), pagefetch.PageVersion{})

	s.clock.now = visitedAt.Add(keptFor)

	if visit, visited := s.records.LastPageVisitOf(t.Context(), pageURL(t)); visited {
		t.Fatalf("a page visit past its retention read back as %+v", visit)
	}
}
