package pagevisitlimits_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitlimits"
)

const (
	keptFor              = time.Hour
	concurrentPageVisits = 8
)

var (
	startedAt         = time.UnixMilli(1_700_000_000_000)
	crawlID           = crawlid.CrawlID{1}
	errEngineRefusing = errors.New("engine refuses writes")
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
		if engine.refusing.Load() {
			return fn(refusingTxn{EngineTxn: etx})
		}
		return fn(etx)
	})
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

type limitsWithin struct {
	storage *vault.Vault
	limits  *pagevisitlimits.PageVisitLimits
}

func (within limitsWithin) TakePageOfHost(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
	maxPages int,
) (bool, error) {
	var took bool
	err := within.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		took, err = within.limits.TakePageOfHost(tx, crawlID, url, maxPages)
		return err
	})
	return took, err
}

func (within limitsWithin) TakeDeferral(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
) (bool, error) {
	var took bool
	err := within.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		took, err = within.limits.TakeDeferral(tx, crawlID, url)
		return err
	})
	return took, err
}

func (within limitsWithin) TakeAttempt(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
) (int, bool, error) {
	var attempt int
	var took bool
	err := within.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		attempt, took, err = within.limits.TakeAttempt(tx, crawlID, url)
		return err
	})
	return attempt, took, err
}

func pageVisitLimitsOn(
	t *testing.T,
	engine vault.Engine,
	clock *settableClock,
) limitsWithin {
	t.Helper()
	storage, err := vault.New(engine, nil, nil, clock)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = storage.Close() })
	limits, err := pagevisitlimits.Open(
		t.Context(),
		storage,
		keptFor,
		pagevisitlimits.MaxPerURL{Deferrals: 2, Attempts: 2},
	)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return limitsWithin{storage: storage, limits: limits}
}

func pageVisitLimits(t *testing.T) (limitsWithin, *settableClock) {
	t.Helper()
	clock := newSettableClock(startedAt)
	return pageVisitLimitsOn(t, memoryvault.OpenEngine(0), clock), clock
}

func limitsOf(t *testing.T) limitsWithin {
	t.Helper()
	limits, _ := pageVisitLimits(t)
	return limits
}

func pageURL(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	return canonicalurltest.CanonicalURLOf(t, "http://host/page")
}

func admitsAnotherPageOfHost(
	t *testing.T,
	limits limitsWithin,
	url canonicalurl.CanonicalURL,
	maxPages int,
) bool {
	t.Helper()
	admitted, err := limits.TakePageOfHost(t.Context(), crawlID, url, maxPages)
	if err != nil {
		t.Fatalf("TakePageOfHost: %v", err)
	}
	return admitted
}

func TestAHostTakesNoMorePagesThanItsProfileAllows(t *testing.T) {
	limits, _ := pageVisitLimits(t)

	for page := range 2 {
		url := canonicalurltest.CanonicalURLOf(t, fmt.Sprintf("http://host/page%d", page))
		if !admitsAnotherPageOfHost(t, limits, url, 2) {
			t.Fatalf("page %d should be within the limit of the host", page+1)
		}
	}

	beyond := canonicalurltest.CanonicalURLOf(t, "http://host/beyond")
	if admitsAnotherPageOfHost(t, limits, beyond, 2) {
		t.Fatal("a page beyond the limit of the host should be refused")
	}
}

func TestEachHostHoldsItsOwnLimitOfPages(t *testing.T) {
	limits, _ := pageVisitLimits(t)
	admitsAnotherPageOfHost(t, limits, pageURL(t), 1)

	otherHost := canonicalurltest.CanonicalURLOf(t, "http://other-host/page")
	if !admitsAnotherPageOfHost(t, limits, otherHost, 1) {
		t.Fatal("a page of another host should be within the limit of its own host")
	}
}

func TestAPageVisitTakesOnePageOfItsHostHoweverOftenItIsRedelivered(t *testing.T) {
	limits, _ := pageVisitLimits(t)
	url := pageURL(t)

	for delivery := range 3 {
		if !admitsAnotherPageOfHost(t, limits, url, 1) {
			t.Fatalf("delivery %d should keep the page the page visit already holds", delivery)
		}
	}

	other := canonicalurltest.CanonicalURLOf(t, "http://host/other")
	if admitsAnotherPageOfHost(t, limits, other, 1) {
		t.Fatal("the redelivered page visit should hold the whole limit of the host")
	}
}

func TestAHostThatAllowsNoPagesAdmitsNone(t *testing.T) {
	if admitsAnotherPageOfHost(t, limitsOf(t), pageURL(t), 0) {
		t.Fatal("a host that allows no pages should refuse the first page")
	}
}

func TestAnUnlimitedHostAdmitsEveryPage(t *testing.T) {
	limits, _ := pageVisitLimits(t)

	for range 3 {
		url := pageURL(t)
		if !admitsAnotherPageOfHost(t, limits, url, yacycrawlcontract.UnlimitedPagesPerHost) {
			t.Fatal("an unlimited host should never refuse a page")
		}
	}
}

func TestTheSameHostCountsSeparatelyUnderAnotherCrawl(t *testing.T) {
	limits, _ := pageVisitLimits(t)
	url := pageURL(t)
	admitsAnotherPageOfHost(t, limits, url, 1)

	other := canonicalurltest.CanonicalURLOf(t, "http://host/other")
	admitted, err := limits.TakePageOfHost(
		t.Context(),
		crawlid.CrawlID{2},
		other,
		1,
	)
	if err != nil {
		t.Fatalf("TakePageOfHost: %v", err)
	}
	if !admitted {
		t.Fatal("another crawl should hold its own limit of the host")
	}
}

func TestOnlyOneOfConcurrentPageVisitsTakesTheLastPageOfAHost(t *testing.T) {
	engine, err := pebblevault.OpenEngine(
		filepath.Join(t.TempDir(), "crawl-state"), 0, pebblevault.MachineLimits{}, nil, nil,
	)
	if err != nil {
		t.Fatalf("OpenEngine: %v", err)
	}
	limits := pageVisitLimitsOn(t, engine, newSettableClock(startedAt))
	admissions := make(chan bool, concurrentPageVisits)

	var taking sync.WaitGroup
	for page := range concurrentPageVisits {
		url := canonicalurltest.CanonicalURLOf(t, fmt.Sprintf("http://host/race%d", page))
		taking.Go(func() {
			admitted, err := limits.TakePageOfHost(
				context.Background(),
				crawlID,
				url,
				1,
			)
			if err != nil {
				t.Errorf("TakePageOfHost: %v", err)
				return
			}
			admissions <- admitted
		})
	}
	taking.Wait()
	close(admissions)

	admitted := 0
	for granted := range admissions {
		if granted {
			admitted++
		}
	}
	if admitted != 1 {
		t.Fatalf("%d concurrent page visits took a host limit of one, want 1", admitted)
	}
}

func TestDeferralsRunOutAtTheirCeiling(t *testing.T) {
	limits, _ := pageVisitLimits(t)
	ctx := t.Context()
	url := pageURL(t)

	for deferral := range 2 {
		admitted, err := limits.TakeDeferral(ctx, crawlID, url)
		if err != nil {
			t.Fatalf("TakeDeferral: %v", err)
		}
		if !admitted {
			t.Fatalf("deferral %d should be within the ceiling", deferral+1)
		}
	}
	admitted, err := limits.TakeDeferral(ctx, crawlID, url)
	if err != nil {
		t.Fatalf("TakeDeferral: %v", err)
	}
	if admitted {
		t.Fatal("a deferral beyond the ceiling should be refused")
	}
}

func TestEachAttemptReportsItsNumberUntilTheCeiling(t *testing.T) {
	limits, _ := pageVisitLimits(t)
	ctx := t.Context()
	url := pageURL(t)

	for want := 1; want <= 2; want++ {
		attempt, admitted, err := limits.TakeAttempt(ctx, crawlID, url)
		if err != nil {
			t.Fatalf("TakeAttempt: %v", err)
		}
		if !admitted || attempt != want {
			t.Fatalf("attempt %d admitted %v, want %d true", attempt, admitted, want)
		}
	}
	if _, admitted, err := limits.TakeAttempt(ctx, crawlID, url); err != nil || admitted {
		t.Fatalf("an attempt beyond the ceiling should be refused, got %v %v", admitted, err)
	}
}

func TestDeferralsAndAttemptsKeepSeparateCeilings(t *testing.T) {
	limits, _ := pageVisitLimits(t)
	ctx := t.Context()
	url := pageURL(t)

	for range 2 {
		if _, err := limits.TakeDeferral(ctx, crawlID, url); err != nil {
			t.Fatalf("TakeDeferral: %v", err)
		}
	}
	_, admitted, err := limits.TakeAttempt(ctx, crawlID, url)
	if err != nil {
		t.Fatalf("TakeAttempt: %v", err)
	}
	if !admitted {
		t.Fatal("spent deferrals should leave the attempts untouched")
	}
}

func TestTakingThePageOfAHostLeavesTheDeferralsUntouched(t *testing.T) {
	limits, _ := pageVisitLimits(t)
	url := pageURL(t)
	admitsAnotherPageOfHost(t, limits, url, 1)

	admitted, err := limits.TakeDeferral(t.Context(), crawlID, url)
	if err != nil {
		t.Fatalf("TakeDeferral: %v", err)
	}
	if !admitted {
		t.Fatal("a page visit holding the page of its host should still defer")
	}
}

func TestAHostTakesPagesAgainOnceItsRetentionPasses(t *testing.T) {
	limits, clock := pageVisitLimits(t)
	if !admitsAnotherPageOfHost(t, limits, pageURL(t), 1) {
		t.Fatal("the first page of the host should be admitted")
	}

	clock.set(startedAt.Add(keptFor))

	other := canonicalurltest.CanonicalURLOf(t, "http://host/other")
	if !admitsAnotherPageOfHost(t, limits, other, 1) {
		t.Fatal("a host past its retention should take pages again")
	}
}

func TestALimitTheStorageRefusesFails(t *testing.T) {
	engine := &refusingEngine{Engine: memoryvault.OpenEngine(0)}
	limits := pageVisitLimitsOn(t, engine, newSettableClock(startedAt))
	engine.refusing.Store(true)

	if _, err := limits.TakePageOfHost(
		t.Context(),
		crawlID,
		pageURL(t),
		1,
	); !errors.Is(
		err,
		errEngineRefusing,
	) {
		t.Fatalf("TakePageOfHost err = %v, want the refusal", err)
	}
	if _, err := limits.TakeDeferral(t.Context(), crawlID, pageURL(t)); !errors.Is(
		err, errEngineRefusing,
	) {
		t.Fatalf("TakeDeferral err = %v, want the refusal", err)
	}
	if _, _, err := limits.TakeAttempt(t.Context(), crawlID, pageURL(t)); !errors.Is(
		err, errEngineRefusing,
	) {
		t.Fatalf("TakeAttempt err = %v, want the refusal", err)
	}
}
