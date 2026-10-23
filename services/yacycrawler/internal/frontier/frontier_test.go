package frontier_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/frontier"
)

const (
	lease             = time.Minute
	concurrentLessees = 8
	contendedVisits   = 32
)

var (
	errNoDueVisit     = errors.New("no page visit is due")
	errEngineRefusing = errors.New("engine refuses")
	firstCrawl        = crawlid.CrawlID{1}
	secondCrawl       = crawlid.CrawlID{2}
)

type clock struct {
	guard sync.Mutex
	now   time.Time
}

func (c *clock) Now() time.Time {
	c.guard.Lock()
	defer c.guard.Unlock()
	return c.now
}

func (c *clock) Sleep(context.Context, time.Duration) error {
	return errNoDueVisit
}

func (c *clock) pass(d time.Duration) {
	c.guard.Lock()
	defer c.guard.Unlock()
	c.now = c.now.Add(d)
}

type crawlLedger struct {
	guard            sync.Mutex
	cancelledOfHost  map[string]uint
	cancelledBatches []uint
	refusing         error
}

func (l *crawlLedger) PageVisitsCancelled(
	_ *vault.Txn,
	crawlID crawlid.CrawlID,
	host string,
	amount uint,
) error {
	l.guard.Lock()
	defer l.guard.Unlock()
	if l.refusing != nil {
		return l.refusing
	}
	l.cancelledOfHost[crawlID.String()+"/"+host] += amount
	l.cancelledBatches = append(l.cancelledBatches, amount)
	return nil
}

type recordedFailures struct {
	guard    sync.Mutex
	leases   []error
	drains   []error
	counts   []error
	ends     []pendingpagevisit.PendingPageVisit
	returns  []pendingpagevisit.PendingPageVisit
	outdated []pendingpagevisit.PendingPageVisit
}

func (r *recordedFailures) FrontierLeaseFailed(_ context.Context, err error) {
	r.guard.Lock()
	defer r.guard.Unlock()
	r.leases = append(r.leases, err)
}

func (r *recordedFailures) FrontierDrainFailed(_ context.Context, err error) {
	r.drains = append(r.drains, err)
}

func (r *recordedFailures) FrontierCountFailed(_ context.Context, err error) {
	r.counts = append(r.counts, err)
}

func (r *recordedFailures) FrontierEndFailed(
	_ context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
	_ error,
) {
	r.ends = append(r.ends, pageVisit)
}

func (r *recordedFailures) FrontierReturnFailed(
	_ context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
	_ time.Duration,
	_ error,
) {
	r.returns = append(r.returns, pageVisit)
}

func (r *recordedFailures) FrontierLeaseOutdated(
	_ context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
) {
	r.outdated = append(r.outdated, pageVisit)
}

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

type frontierScene struct {
	storage  *vault.Vault
	engine   *refusingEngine
	clock    *clock
	ledger   *crawlLedger
	failures *recordedFailures
	visits   *frontier.Frontier
}

func newFrontier(t *testing.T) frontierScene {
	t.Helper()
	return newFrontierOn(t, memoryvault.OpenEngine(0))
}

func newFrontierOn(t *testing.T, engine vault.Engine) frontierScene {
	t.Helper()
	refusing := &refusingEngine{Engine: engine}
	visitClock := &clock{now: time.Unix(1_000_000, 0)}
	storage, err := vault.New(refusing, nil, nil, visitClock)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = storage.Close() })
	ledger := &crawlLedger{cancelledOfHost: map[string]uint{}}
	failures := &recordedFailures{}
	visits, err := frontier.Open(
		t.Context(), storage, ledger, lease, visitClock, failures,
	)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return frontierScene{
		storage: storage, engine: refusing, clock: visitClock, ledger: ledger, failures: failures,
		visits: visits,
	}
}

func pageVisitTo(t *testing.T, rawURL string) pendingpagevisit.PendingPageVisit {
	t.Helper()
	return pageVisitOfCrawl(t, firstCrawl, rawURL)
}

func pageVisitOfCrawl(
	t *testing.T,
	crawlID crawlid.CrawlID,
	rawURL string,
) pendingpagevisit.PendingPageVisit {
	t.Helper()
	return pendingpagevisit.PendingPageVisit{
		CrawlID: crawlID,
		URL:     canonicalurltest.CanonicalURLOf(t, rawURL),
		Depth:   1,
	}
}

func (f frontierScene) update(t *testing.T, fn func(tx *vault.Txn) error) {
	t.Helper()
	if err := f.storage.Update(t.Context(), fn); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (f frontierScene) change(
	t *testing.T,
	change func(tx *vault.Txn, crawlID crawlid.CrawlID) error,
	crawlID crawlid.CrawlID,
) {
	t.Helper()
	f.update(t, func(tx *vault.Txn) error { return change(tx, crawlID) })
}

func (f frontierScene) add(t *testing.T, pageVisits ...pendingpagevisit.PendingPageVisit) {
	t.Helper()
	f.update(t, func(tx *vault.Txn) error { return f.visits.Add(tx, pageVisits) })
}

func (f frontierScene) next(t *testing.T) pendingpagevisit.LeasedPageVisit {
	t.Helper()
	leasedPageVisit, err := f.visits.Next(t.Context())
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	return leasedPageVisit
}

func (f frontierScene) endNext(t *testing.T) pendingpagevisit.PendingPageVisit {
	t.Helper()
	leasedPageVisit := f.next(t)
	f.visits.End(t.Context(), leasedPageVisit)
	return leasedPageVisit.PageVisit
}

func (f frontierScene) requireNoDueVisit(t *testing.T) {
	t.Helper()
	if leasedPageVisit, err := f.visits.Next(t.Context()); !errors.Is(err, errNoDueVisit) {
		t.Fatalf("Next = %v, %v; want no due page visit", leasedPageVisit, err)
	}
}

func (f frontierScene) drainCancelled(t *testing.T) {
	t.Helper()
	if err := f.visits.DrainCancelledCrawls(t.Context()); err != nil {
		t.Fatalf("DrainCancelledCrawls: %v", err)
	}
}

func (f frontierScene) amount(t *testing.T) int64 {
	t.Helper()
	amount, counted := f.visits.AmountPending(t.Context())
	if !counted {
		t.Fatal("AmountPending did not count")
	}
	return amount
}

func TestAPendingPageVisitIsLeasedWithItsDepthUntilTheLeaseExpires(t *testing.T) {
	f := newFrontier(t)
	pageVisit := pageVisitTo(t, "http://host/page").LinkedPageVisitTo(
		canonicalurltest.CanonicalURLOf(t, "http://host/deeper"),
	)
	f.add(t, pageVisit)

	leasedPageVisit := f.next(t)

	if leasedPageVisit.PageVisit != pageVisit ||
		!leasedPageVisit.LeasedUntil.Equal(f.clock.Now().Add(lease)) {
		t.Fatalf("leased %+v, want %+v until %v", leasedPageVisit, pageVisit,
			f.clock.Now().Add(lease))
	}
}

func TestAPageVisitAddedTwiceIsTakenOnce(t *testing.T) {
	f := newFrontier(t)
	pageVisit := pageVisitTo(t, "http://host/page")

	f.add(t, pageVisit, pageVisit)
	f.add(t, pageVisit)

	f.endNext(t)
	f.requireNoDueVisit(t)
}

func TestATakenPageVisitComesBackOnlyOnceItsLeaseExpires(t *testing.T) {
	f := newFrontier(t)
	f.add(t, pageVisitTo(t, "http://host/page"))
	f.next(t)

	f.requireNoDueVisit(t)
	f.clock.pass(lease)

	f.next(t)
}

func TestAnEndedPageVisitNeverComesBackNorIsAddedAgain(t *testing.T) {
	f := newFrontier(t)
	pageVisit := pageVisitTo(t, "http://host/page")
	f.add(t, pageVisit)
	f.endNext(t)

	f.add(t, pageVisit)
	f.clock.pass(2 * lease)

	f.requireNoDueVisit(t)
	if amount := f.amount(t); amount != 0 {
		t.Fatalf("pending %d, want none", amount)
	}
}

func TestAPageVisitReturnedAfterADelayIsDueOnlyOnceTheDelayPasses(t *testing.T) {
	f := newFrontier(t)
	f.add(t, pageVisitTo(t, "http://host/page"))
	f.visits.Return(t.Context(), f.next(t), 2*lease)

	f.clock.pass(lease)
	f.requireNoDueVisit(t)
	f.clock.pass(lease)

	f.next(t)
}

func TestAReturnedPageVisitComesBackSoonerThanItsLease(t *testing.T) {
	f := newFrontier(t)
	f.add(t, pageVisitTo(t, "http://host/page"))
	f.visits.Return(t.Context(), f.next(t), lease/4)

	f.clock.pass(lease / 2)

	f.next(t)
}

func TestAPageVisitReturnedWhileItsCrawlIsHeldIsLeasedOnlyOnceReleased(t *testing.T) {
	f := newFrontier(t)
	returned := pageVisitTo(t, "http://host/page")
	f.add(t, returned)
	leased := f.next(t)
	f.change(t, f.visits.Hold, firstCrawl)

	f.visits.Return(t.Context(), leased, 0)
	f.requireNoDueVisit(t)
	f.change(t, f.visits.Release, firstCrawl)

	if again := f.next(t).PageVisit; again.URL != returned.URL {
		t.Fatalf("leased %+v after the release, want the returned visit", again)
	}
}

func TestAnEndUnderAnExpiredLeaseChangesNothing(t *testing.T) {
	f := newFrontier(t)
	f.add(t, pageVisitTo(t, "http://host/page"))
	first := f.next(t)
	f.clock.pass(lease)
	second := f.next(t)

	f.visits.Return(t.Context(), first, time.Hour)
	f.visits.End(t.Context(), first)

	if len(f.failures.outdated) != 2 || f.amount(t) != 1 {
		t.Fatalf("outdated %v pending %d, want both ends outdated and the visit pending",
			f.failures.outdated, f.amount(t))
	}
	f.visits.End(t.Context(), second)
	f.clock.pass(2 * lease)
	f.requireNoDueVisit(t)
}

func TestPageVisitsAreTakenInTheOrderTheyFallDueAcrossCrawls(t *testing.T) {
	f := newFrontier(t)
	f.add(t, pageVisitOfCrawl(t, firstCrawl, "http://host/first"))
	f.visits.Return(t.Context(), f.next(t), 3*time.Second)
	f.clock.pass(time.Second)
	f.add(t, pageVisitOfCrawl(t, secondCrawl, "http://host/second"))
	f.clock.pass(time.Hour)

	got := []crawlid.CrawlID{f.endNext(t).CrawlID, f.endNext(t).CrawlID}

	if !slices.Equal(got, []crawlid.CrawlID{secondCrawl, firstCrawl}) {
		t.Fatalf("took %v, want the earliest due first", got)
	}
}

func TestAHeldCrawlKeepsItsVisitsAndLendsNoneUntilReleased(t *testing.T) {
	f := newFrontier(t)
	f.add(t, pageVisitTo(t, "http://host/a"))
	f.change(t, f.visits.Hold, firstCrawl)

	f.add(t, pageVisitTo(t, "http://host/b"))
	f.requireNoDueVisit(t)
	if amount := f.amount(t); amount != 2 {
		t.Fatalf("pending %d, want both visits kept", amount)
	}
	f.change(t, f.visits.Release, firstCrawl)

	f.endNext(t)
	f.endNext(t)
}

func TestThePendingPageVisitsAreCountedPerCrawlAndHost(t *testing.T) {
	f := newFrontier(t)
	f.add(
		t,
		pageVisitTo(t, "http://a/1"),
		pageVisitTo(t, "http://a/2"),
		pageVisitTo(t, "http://b/1"),
		pageVisitOfCrawl(t, secondCrawl, "http://a/1"),
	)
	var pendingOfFirst int
	pendingOfHosts := map[string]int{}

	f.update(t, func(tx *vault.Txn) error {
		var err error
		if pendingOfFirst, err = f.visits.AmountPendingOf(tx, firstCrawl); err != nil {
			return err
		}
		return f.visits.AmountsPendingByHostOf(tx, firstCrawl, func(host string, amount int) {
			pendingOfHosts[host] = amount
		})
	})

	if pendingOfFirst != 3 || pendingOfHosts["a"] != 2 || pendingOfHosts["b"] != 1 ||
		f.amount(t) != 4 {
		t.Fatalf("pending by first %d, of hosts %v, every %d; want 3, a=2 b=1, 4", pendingOfFirst,
			pendingOfHosts, f.amount(t))
	}
}

func TestAPageVisitTakenIsCountedUntilEnded(t *testing.T) {
	f := newFrontier(t)
	f.add(t, pageVisitTo(t, "http://host/a"), pageVisitTo(t, "http://host/b"))
	f.next(t)

	if amount := f.amount(t); amount != 2 {
		t.Fatalf("pending %d, want 2", amount)
	}
}

func TestACancelledCrawlTakesNoVisitAndDropsThePendingOnesLeasedOrNot(t *testing.T) {
	f := newFrontier(t)
	f.add(t, pageVisitTo(t, "http://a/1"), pageVisitTo(t, "http://b/1"),
		pageVisitTo(t, "http://b/2"))
	leased := f.next(t)

	f.change(t, f.visits.Cancel, firstCrawl)
	f.add(t, pageVisitTo(t, "http://c/1"))
	f.requireNoDueVisit(t)
	f.drainCancelled(t)
	f.visits.End(t.Context(), leased)

	drained := f.ledger.cancelledOfHost
	if drained[firstCrawl.String()+"/a"]+drained[firstCrawl.String()+"/b"] != 3 ||
		f.amount(t) != 0 || len(f.failures.outdated) != 1 {
		t.Fatalf("drained %v, pending %d, outdated %v; want all three cancelled", drained,
			f.amount(t), f.failures.outdated)
	}
}

func TestACancelDropsThePendingVisitsInBatchesBesideTheLeases(t *testing.T) {
	f := newFrontier(t)
	const pending = 150
	pageVisits := make([]pendingpagevisit.PendingPageVisit, 0, pending)
	for visit := range pending {
		pageVisits = append(pageVisits, pageVisitTo(t, fmt.Sprintf("http://host/%d", visit)))
	}
	f.add(t, pageVisits...)
	f.change(t, f.visits.Cancel, firstCrawl)
	f.add(t, pageVisitOfCrawl(t, secondCrawl, "http://other/"))

	if leased := f.next(t).PageVisit; leased.CrawlID != secondCrawl {
		t.Fatalf("leased %+v, want the visit of the open crawl", leased)
	}
	if amount := f.amount(t); amount != pending+1 {
		t.Fatalf("pending %d after a lease, want no visit dropped by leasing", amount)
	}
	f.drainCancelled(t)

	if !slices.Equal(f.ledger.cancelledBatches, []uint{100, 50}) || f.amount(t) != 1 {
		t.Fatalf("batches %v, pending %d; want two batches and the open crawl's visit",
			f.ledger.cancelledBatches, f.amount(t))
	}
}

func TestADrainEmptiesEveryCancelledCrawlBeyondTheOnesItPicksAmong(t *testing.T) {
	f := newFrontier(t)
	const cancelledCrawls = 12
	for crawl := range cancelledCrawls {
		crawlID := crawlid.CrawlID{byte(crawl + 10)}
		f.add(t,
			pageVisitOfCrawl(t, crawlID, "http://a/"+strconv.Itoa(crawl)),
			pageVisitOfCrawl(t, crawlID, "http://b/"+strconv.Itoa(crawl)),
		)
		f.change(t, f.visits.Cancel, crawlID)
	}

	f.drainCancelled(t)

	if f.amount(t) != 0 || len(f.ledger.cancelledBatches) != 2*cancelledCrawls {
		t.Fatalf("pending %d, batches %v; want both hosts of every cancelled crawl drained once",
			f.amount(t), f.ledger.cancelledBatches)
	}
}

func TestACancelDropsVisitsWhileEveryPageVisitIsPaused(t *testing.T) {
	f := newFrontier(t)
	f.add(t, pageVisitTo(t, "http://host/a"))
	if err := f.visits.Pause(t.Context()); err != nil {
		t.Fatalf("Pause: %v", err)
	}

	f.change(t, f.visits.Cancel, firstCrawl)
	f.drainCancelled(t)

	if f.amount(t) != 0 {
		t.Fatalf("pending %d, want the cancelled crawl drained", f.amount(t))
	}
}

func TestCancellingAHeldCrawlDropsItsVisits(t *testing.T) {
	f := newFrontier(t)
	f.add(t, pageVisitTo(t, "http://host/a"))
	f.change(t, f.visits.Hold, firstCrawl)

	f.change(t, f.visits.Cancel, firstCrawl)
	f.change(t, f.visits.Release, firstCrawl)
	f.requireNoDueVisit(t)
	f.drainCancelled(t)

	if f.amount(t) != 0 || len(f.ledger.cancelledBatches) != 1 {
		t.Fatalf("pending %d, batches %v; want the held crawl drained",
			f.amount(t), f.ledger.cancelledBatches)
	}
}

func TestAClosedCrawlTakesNoVisitAndIsNotDrained(t *testing.T) {
	f := newFrontier(t)

	f.change(t, f.visits.Close, firstCrawl)
	f.change(t, f.visits.Hold, firstCrawl)
	f.change(t, f.visits.Cancel, firstCrawl)
	f.add(t, pageVisitTo(t, "http://host/a"))
	f.requireNoDueVisit(t)
	f.drainCancelled(t)

	if f.amount(t) != 0 || len(f.ledger.cancelledOfHost) != 0 {
		t.Fatalf("pending %d, cancelled %v; want nothing",
			f.amount(t), f.ledger.cancelledOfHost)
	}
}

func TestAForgottenCrawlKeepsNothingAndItsVisitsArePendingAgain(t *testing.T) {
	f := newFrontier(t)
	ended, pending := pageVisitTo(t, "http://host/a"), pageVisitTo(t, "http://host/b")
	f.add(t, ended, pending)
	f.endNext(t)
	f.change(t, f.visits.Close, firstCrawl)

	f.change(t, f.visits.Forget, firstCrawl)

	if f.amount(t) != 0 {
		t.Fatalf("pending %d after forgetting, want none", f.amount(t))
	}
	f.add(t, ended, pending)
	if f.amount(t) != 2 {
		t.Fatalf("pending %d, want both visits pending again", f.amount(t))
	}
	f.endNext(t)
}

func TestACancelTheLedgerRefusesIsReportedAndTriedAgain(t *testing.T) {
	f := newFrontier(t)
	f.add(t, pageVisitTo(t, "http://host/a"))
	f.change(t, f.visits.Cancel, firstCrawl)
	f.ledger.refusing = errEngineRefusing

	f.drainCancelled(t)
	f.ledger.refusing = nil
	f.drainCancelled(t)

	if len(f.failures.drains) != 1 || f.amount(t) != 0 {
		t.Fatalf("drain failures %v, pending %d; want one failure, then drained",
			f.failures.drains, f.amount(t))
	}
}

func TestThePausedFrontierLendsNoVisitUntilResumed(t *testing.T) {
	f := newFrontier(t)
	f.add(t, pageVisitTo(t, "http://host/a"))

	if err := f.visits.Pause(t.Context()); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	paused, err := f.visits.IsPaused(t.Context())
	if err != nil || !paused {
		t.Fatalf("IsPaused = %v, %v; want paused", paused, err)
	}
	f.requireNoDueVisit(t)
	if err := f.visits.Resume(t.Context()); err != nil {
		t.Fatalf("Resume: %v", err)
	}

	f.next(t)
}

func TestFrontierOperationsTheStorageRefusesAreReportedOrFail(t *testing.T) {
	f := newFrontier(t)
	f.add(t, pageVisitTo(t, "http://host/a"))
	leasedPageVisit := f.next(t)
	f.engine.refusing = true

	f.visits.End(t.Context(), leasedPageVisit)
	f.visits.Return(t.Context(), leasedPageVisit, time.Second)
	_, counted := f.visits.AmountPending(t.Context())
	f.requireNoDueVisit(t)
	f.drainCancelled(t)
	pauseErr := f.visits.Pause(t.Context())
	resumeErr := f.visits.Resume(t.Context())
	_, pausedErr := f.visits.IsPaused(t.Context())

	if len(f.failures.ends) != 1 || len(f.failures.returns) != 1 || counted ||
		len(f.failures.counts) != 1 || len(f.failures.leases) != 1 ||
		len(f.failures.drains) != 1 {
		t.Fatalf("failures = %+v counted %v, want each refusal reported once", f.failures, counted)
	}
	for name, err := range map[string]error{"pause": pauseErr, "resume": resumeErr, "paused": pausedErr} {
		if !errors.Is(err, errEngineRefusing) {
			t.Errorf("%s = %v, want the refusal", name, err)
		}
	}
}

func TestConcurrentLeasesTakeEachPendingPageVisitOnce(t *testing.T) {
	engine, err := pebblevault.OpenEngine(
		filepath.Join(t.TempDir(), "crawl-state"), 0, pebblevault.MachineLimits{}, nil, nil,
	)
	if err != nil {
		t.Fatalf("OpenEngine: %v", err)
	}
	f := newFrontierOn(t, engine)
	pageVisits := make([]pendingpagevisit.PendingPageVisit, 0, contendedVisits)
	for visit := range contendedVisits {
		pageVisits = append(pageVisits, pageVisitTo(t, fmt.Sprintf("http://host%d/", visit)))
	}
	f.add(t, pageVisits...)

	taken := make(chan string, contendedVisits)
	var lessees sync.WaitGroup
	for range concurrentLessees {
		lessees.Go(func() {
			for {
				leasedPageVisit, err := f.visits.Next(t.Context())
				if err != nil {
					return
				}
				f.visits.End(t.Context(), leasedPageVisit)
				taken <- leasedPageVisit.PageVisit.URL.String()
			}
		})
	}
	lessees.Wait()
	close(taken)

	seen := map[string]int{}
	for url := range taken {
		seen[url]++
	}
	if len(seen) != contendedVisits {
		t.Fatalf("took %d visits, want %d", len(seen), contendedVisits)
	}
	for url, times := range seen {
		if times != 1 {
			t.Fatalf("took %s %d times, want once", url, times)
		}
	}
}
