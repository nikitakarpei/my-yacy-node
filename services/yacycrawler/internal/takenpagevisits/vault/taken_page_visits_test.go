package vault_test

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	takenpagevisitsvault "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/takenpagevisits/vault"
)

const (
	taker      = "YACY_CRAWL_FRONTIER/1"
	keptFor    = time.Hour
	takerCount = 8
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
	if engine.refusing.Load() {
		return errEngineRefusing
	}
	return engine.Engine.Update(ctx, fn)
}

func takenPageVisitsOn(
	t *testing.T,
	engine vault.Engine,
	clock *settableClock,
) *takenpagevisitsvault.TakenPageVisits {
	t.Helper()
	storage, err := vault.New(engine, nil, nil, clock)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = storage.Close() })
	visits, err := takenpagevisitsvault.New(
		t.Context(),
		storage,
		keptFor,
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return visits
}

func takenPageVisits(t *testing.T) (*takenpagevisitsvault.TakenPageVisits, *settableClock) {
	t.Helper()
	clock := newSettableClock(startedAt)
	return takenPageVisitsOn(t, memoryvault.OpenEngine(0), clock), clock
}

func pageURL(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	return canonicalurltest.CanonicalURLOf(t, "http://host/page")
}

func takePageVisit(
	t *testing.T,
	visits *takenpagevisitsvault.TakenPageVisits,
	url canonicalurl.CanonicalURL,
	taker string,
) bool {
	t.Helper()
	took, err := visits.TakePageVisit(t.Context(), crawlID, url, taker)
	if err != nil {
		t.Fatalf("TakePageVisit: %v", err)
	}
	return took
}

func TestTheFirstTakerTakesThePageVisitAndTheNextDoesNot(t *testing.T) {
	visits, _ := takenPageVisits(t)
	url := pageURL(t)

	if !takePageVisit(t, visits, url, taker) {
		t.Fatal("the first taker should take the page visit")
	}
	if takePageVisit(t, visits, url, "another taker") {
		t.Fatal("a second taker should find the page visit taken")
	}
}

func TestTheTakerOfAPageVisitTakesItAgainOnRedelivery(t *testing.T) {
	visits, _ := takenPageVisits(t)
	url := pageURL(t)
	takePageVisit(t, visits, url, taker)

	if !takePageVisit(t, visits, url, taker) {
		t.Fatal("the taker should keep the page visit it already took")
	}
}

func TestTheSameURLIsTakableAgainUnderAnotherCrawl(t *testing.T) {
	visits, _ := takenPageVisits(t)
	url := pageURL(t)
	takePageVisit(t, visits, url, taker)

	took, err := visits.TakePageVisit(t.Context(), crawlid.CrawlID{2}, url, "another taker")
	if err != nil {
		t.Fatalf("TakePageVisit: %v", err)
	}
	if !took {
		t.Fatal("another crawl should take the same url")
	}
}

func TestAPageVisitIsTakableAgainOnceItsRetentionPasses(t *testing.T) {
	visits, clock := takenPageVisits(t)
	url := pageURL(t)
	takePageVisit(t, visits, url, taker)

	clock.set(startedAt.Add(keptFor))

	if !takePageVisit(t, visits, url, "another taker") {
		t.Fatal("a page visit past its retention should be takable again")
	}
}

func TestATakeTheStorageRefusesFails(t *testing.T) {
	engine := &refusingEngine{Engine: memoryvault.OpenEngine(0)}
	visits := takenPageVisitsOn(t, engine, newSettableClock(startedAt))
	engine.refusing.Store(true)

	if _, err := visits.TakePageVisit(t.Context(), crawlID, pageURL(t), taker); !errors.Is(
		err, errEngineRefusing,
	) {
		t.Fatalf("TakePageVisit err = %v, want the refusal", err)
	}
}

func TestConcurrentTakersOfOnePageVisitLeaveOneWinner(t *testing.T) {
	engine, err := pebblevault.OpenEngine(
		filepath.Join(t.TempDir(), "crawl-state"), 0, pebblevault.MachineLimits{}, nil, nil,
	)
	if err != nil {
		t.Fatalf("OpenEngine: %v", err)
	}
	visits := takenPageVisitsOn(t, engine, newSettableClock(startedAt))
	url := pageURL(t)

	took := make([]bool, takerCount)
	var taking sync.WaitGroup
	for position := range takerCount {
		taking.Go(func() {
			took[position], _ = visits.TakePageVisit(
				context.Background(), crawlID, url, strconv.Itoa(position),
			)
		})
	}
	taking.Wait()

	winners := 0
	for _, won := range took {
		if won {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("%d takers took one page visit, want 1", winners)
	}
}
