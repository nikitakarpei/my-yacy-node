package frontier_test

import (
	"bytes"
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
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
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
	owingNothing     []crawlid.CrawlID
	cancelledOfHost  map[string]uint
	cancelledBatches []uint
	refusing         error
}

func (l *crawlLedger) CrawlOwesNothing(_ *vault.Txn, crawlID crawlid.CrawlID) error {
	l.guard.Lock()
	defer l.guard.Unlock()
	if l.refusing != nil {
		return l.refusing
	}
	l.owingNothing = append(l.owingNothing, crawlID)
	return nil
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
	guard           sync.Mutex
	leases          []error
	drains          []error
	counts          []error
	acknowledgments []string
	returns         []string
	outdated        []string
	notSettled      []crawlid.CrawlID
}

func (r *recordedFailures) FrontierLeaseFailed(_ context.Context, err error) {
	r.guard.Lock()
	defer r.guard.Unlock()
	r.leases = append(r.leases, err)
}

func (r *recordedFailures) FrontierDrainFailed(_ context.Context, err error) {
	r.drains = append(r.drains, err)
}

func (r *recordedFailures) FrontierCrawlNotSettled(
	_ context.Context,
	crawlID crawlid.CrawlID,
	_ error,
) {
	r.notSettled = append(r.notSettled, crawlID)
}

func (r *recordedFailures) FrontierCountFailed(_ context.Context, err error) {
	r.counts = append(r.counts, err)
}

func (r *recordedFailures) FrontierAcknowledgeFailed(_ context.Context, identity string, _ error) {
	r.acknowledgments = append(r.acknowledgments, identity)
}

func (r *recordedFailures) FrontierReturnFailed(
	_ context.Context,
	identity string,
	_ time.Duration,
	_ error,
) {
	r.returns = append(r.returns, identity)
}

func (r *recordedFailures) FrontierLeaseOutdated(_ context.Context, identity string) {
	r.outdated = append(r.outdated, identity)
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
		t.Context(), storage, ledger, ledger, lease, visitClock, failures,
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

func (f frontierScene) owe(t *testing.T, pageVisits ...pendingpagevisit.PendingPageVisit) {
	t.Helper()
	f.update(t, func(tx *vault.Txn) error { return f.visits.Owe(tx, pageVisits) })
}

func (f frontierScene) next(t *testing.T) pullintake.PendingMessage {
	t.Helper()
	message, err := f.visits.Next(t.Context())
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	return message
}

func (f frontierScene) settleNext(t *testing.T) pendingpagevisit.PendingPageVisit {
	t.Helper()
	message := f.next(t)
	message.Acknowledge(t.Context())
	return pageVisitOf(t, message)
}

func (f frontierScene) requireNoDueVisit(t *testing.T) {
	t.Helper()
	if message, err := f.visits.Next(t.Context()); !errors.Is(err, errNoDueVisit) {
		t.Fatalf("Next = %v, %v; want no due page visit", message, err)
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
	amount, counted := f.visits.AmountOfPendingPageVisits(t.Context())
	if !counted {
		t.Fatal("AmountOfPendingPageVisits did not count")
	}
	return amount
}

func pageVisitOf(
	t *testing.T,
	message pullintake.PendingMessage,
) pendingpagevisit.PendingPageVisit {
	t.Helper()
	pageVisit, err := pendingpagevisit.UnmarshalPendingPageVisit(message.Body())
	if err != nil {
		t.Fatalf("UnmarshalPendingPageVisit: %v", err)
	}
	return pageVisit
}

func TestAnOwedPageVisitIsTakenWithItsArrival(t *testing.T) {
	f := newFrontier(t)
	pageVisit := pageVisitTo(t, "http://host/page")
	f.owe(t, pageVisit)
	arrival := f.clock.Now()
	f.clock.pass(time.Second)

	message := f.next(t)

	if got := pageVisitOf(t, message); got != pageVisit {
		t.Fatalf("page visit = %+v, want %+v", got, pageVisit)
	}
	if !message.Arrival().Equal(arrival) || message.Identity() == "" {
		t.Fatalf("message arrived %v as %q, want %v with an identity", message.Arrival(),
			message.Identity(), arrival)
	}
}

func TestAPageVisitOwedTwiceIsTakenOnce(t *testing.T) {
	f := newFrontier(t)
	pageVisit := pageVisitTo(t, "http://host/page")

	f.owe(t, pageVisit, pageVisit)
	f.owe(t, pageVisit)

	f.settleNext(t)
	f.requireNoDueVisit(t)
}

func TestATakenPageVisitComesBackOnlyOnceItsLeaseEnds(t *testing.T) {
	f := newFrontier(t)
	f.owe(t, pageVisitTo(t, "http://host/page"))
	f.next(t)

	f.requireNoDueVisit(t)
	f.clock.pass(lease)

	f.next(t)
}

func TestAnAcknowledgedPageVisitNeverComesBackNorIsOwedAgain(t *testing.T) {
	f := newFrontier(t)
	pageVisit := pageVisitTo(t, "http://host/page")
	f.owe(t, pageVisit)
	f.settleNext(t)

	f.owe(t, pageVisit)
	f.clock.pass(2 * lease)

	f.requireNoDueVisit(t)
	if amount := f.amount(t); amount != 0 {
		t.Fatalf("owed %d, want none", amount)
	}
}

func TestAPageVisitReturnedAfterADelayIsDueOnlyOnceTheDelayPasses(t *testing.T) {
	f := newFrontier(t)
	f.owe(t, pageVisitTo(t, "http://host/page"))
	f.next(t).ReturnAfter(t.Context(), 2*lease)

	f.clock.pass(lease)
	f.requireNoDueVisit(t)
	f.clock.pass(lease)

	f.next(t)
}

func TestAReturnedPageVisitComesBackSoonerThanItsLease(t *testing.T) {
	f := newFrontier(t)
	f.owe(t, pageVisitTo(t, "http://host/page"))
	f.next(t).Return(t.Context())

	f.clock.pass(lease / 2)

	f.next(t)
}

func TestASettlementUnderAnEndedLeaseChangesNothing(t *testing.T) {
	f := newFrontier(t)
	f.owe(t, pageVisitTo(t, "http://host/page"))
	first := f.next(t)
	f.clock.pass(lease)
	second := f.next(t)

	first.ReturnAfter(t.Context(), time.Hour)
	first.Acknowledge(t.Context())

	if len(f.failures.outdated) != 2 || f.amount(t) != 1 {
		t.Fatalf("outdated %v owed %d, want both settlements outdated and the visit owed",
			f.failures.outdated, f.amount(t))
	}
	second.Acknowledge(t.Context())
	f.clock.pass(2 * lease)
	f.requireNoDueVisit(t)
}

func TestPageVisitsAreTakenInTheOrderTheyFallDueAcrossCrawls(t *testing.T) {
	f := newFrontier(t)
	f.owe(t, pageVisitOfCrawl(t, firstCrawl, "http://host/first"))
	f.next(t).ReturnAfter(t.Context(), 3*time.Second)
	f.clock.pass(time.Second)
	f.owe(t, pageVisitOfCrawl(t, secondCrawl, "http://host/second"))
	f.clock.pass(time.Hour)

	got := []crawlid.CrawlID{f.settleNext(t).CrawlID, f.settleNext(t).CrawlID}

	if !slices.Equal(got, []crawlid.CrawlID{secondCrawl, firstCrawl}) {
		t.Fatalf("took %v, want the earliest due first", got)
	}
}

func TestAHeldCrawlKeepsItsVisitsAndLendsNoneUntilReleased(t *testing.T) {
	f := newFrontier(t)
	f.owe(t, pageVisitTo(t, "http://host/a"))
	f.change(t, f.visits.Hold, firstCrawl)

	f.owe(t, pageVisitTo(t, "http://host/b"))
	f.requireNoDueVisit(t)
	if amount := f.amount(t); amount != 2 {
		t.Fatalf("owed %d, want both visits kept", amount)
	}
	f.change(t, f.visits.Release, firstCrawl)

	f.settleNext(t)
	f.settleNext(t)
}

func TestTheOwedPageVisitsAreCountedPerCrawlAndHost(t *testing.T) {
	f := newFrontier(t)
	f.owe(
		t,
		pageVisitTo(t, "http://a/1"),
		pageVisitTo(t, "http://a/2"),
		pageVisitTo(t, "http://b/1"),
		pageVisitOfCrawl(t, secondCrawl, "http://a/1"),
	)
	var owedByFirst int
	owedOfHosts := map[string]int{}

	f.update(t, func(tx *vault.Txn) error {
		var err error
		if owedByFirst, err = f.visits.AmountOwedBy(tx, firstCrawl); err != nil {
			return err
		}
		return f.visits.AmountsOwedByHostsOf(tx, firstCrawl, func(host string, amount int) {
			owedOfHosts[host] = amount
		})
	})

	if owedByFirst != 3 || owedOfHosts["a"] != 2 || owedOfHosts["b"] != 1 || f.amount(t) != 4 {
		t.Fatalf("owed by first %d, of hosts %v, every %d; want 3, a=2 b=1, 4", owedByFirst,
			owedOfHosts, f.amount(t))
	}
}

func TestAPageVisitTakenIsCountedUntilAcknowledged(t *testing.T) {
	f := newFrontier(t)
	f.owe(t, pageVisitTo(t, "http://host/a"), pageVisitTo(t, "http://host/b"))
	f.next(t)

	if amount := f.amount(t); amount != 2 {
		t.Fatalf("owed %d, want 2", amount)
	}
}

func TestTheLastAcknowledgedVisitReportsItsCrawlOwesNothing(t *testing.T) {
	f := newFrontier(t)
	f.owe(t, pageVisitTo(t, "http://host/a"), pageVisitTo(t, "http://host/b"))

	f.settleNext(t)
	owingNothingEarly := len(f.ledger.owingNothing)
	f.settleNext(t)

	if owingNothingEarly != 0 ||
		!slices.Equal(f.ledger.owingNothing, []crawlid.CrawlID{firstCrawl}) {
		t.Fatalf("owing nothing after the first %d, then %v; want only after the last",
			owingNothingEarly, f.ledger.owingNothing)
	}
}

func TestACancelledCrawlTakesNoVisitAndDropsTheOwedOnesLeasedOrNot(t *testing.T) {
	f := newFrontier(t)
	f.owe(t, pageVisitTo(t, "http://a/1"), pageVisitTo(t, "http://b/1"),
		pageVisitTo(t, "http://b/2"))
	leased := f.next(t)

	f.change(t, f.visits.Cancel, firstCrawl)
	f.owe(t, pageVisitTo(t, "http://c/1"))
	f.requireNoDueVisit(t)
	f.drainCancelled(t)
	leased.Acknowledge(t.Context())

	drained := f.ledger.cancelledOfHost
	if drained[firstCrawl.String()+"/a"]+drained[firstCrawl.String()+"/b"] != 3 ||
		f.amount(t) != 0 || len(f.failures.outdated) != 1 {
		t.Fatalf("drained %v, owed %d, outdated %v; want all three cancelled", drained,
			f.amount(t), f.failures.outdated)
	}
	if !slices.Equal(f.ledger.owingNothing, []crawlid.CrawlID{firstCrawl}) {
		t.Fatalf("owing nothing %v, want the cancelled crawl once", f.ledger.owingNothing)
	}
}

func TestACancelDropsTheOwedVisitsInBatchesBesideTheLeases(t *testing.T) {
	f := newFrontier(t)
	const owed = 150
	pageVisits := make([]pendingpagevisit.PendingPageVisit, 0, owed)
	for visit := range owed {
		pageVisits = append(pageVisits, pageVisitTo(t, fmt.Sprintf("http://host/%d", visit)))
	}
	f.owe(t, pageVisits...)
	f.change(t, f.visits.Cancel, firstCrawl)
	f.owe(t, pageVisitOfCrawl(t, secondCrawl, "http://other/"))

	if leased := pageVisitOf(t, f.next(t)); leased.CrawlID != secondCrawl {
		t.Fatalf("leased %+v, want the visit of the open crawl", leased)
	}
	if amount := f.amount(t); amount != owed+1 {
		t.Fatalf("owed %d after a lease, want no visit dropped by leasing", amount)
	}
	f.drainCancelled(t)

	if !slices.Equal(f.ledger.cancelledBatches, []uint{100, 50}) || f.amount(t) != 1 ||
		!slices.Equal(f.ledger.owingNothing, []crawlid.CrawlID{firstCrawl}) {
		t.Fatalf("batches %v, owed %d, owing nothing %v; want two batches and the crawl settled",
			f.ledger.cancelledBatches, f.amount(t), f.ledger.owingNothing)
	}
}

func TestADrainEmptiesEveryCancelledCrawlBeyondTheOnesItPicksAmong(t *testing.T) {
	f := newFrontier(t)
	const cancelledCrawls = 12
	cancelled := make([]crawlid.CrawlID, 0, cancelledCrawls)
	for crawl := range cancelledCrawls {
		crawlID := crawlid.CrawlID{byte(crawl + 10)}
		cancelled = append(cancelled, crawlID)
		f.owe(t,
			pageVisitOfCrawl(t, crawlID, "http://a/"+strconv.Itoa(crawl)),
			pageVisitOfCrawl(t, crawlID, "http://b/"+strconv.Itoa(crawl)),
		)
		f.change(t, f.visits.Cancel, crawlID)
	}

	f.drainCancelled(t)

	slices.SortFunc(
		f.ledger.owingNothing,
		func(a, b crawlid.CrawlID) int { return bytes.Compare(a[:], b[:]) },
	)
	if f.amount(t) != 0 || !slices.Equal(f.ledger.owingNothing, cancelled) {
		t.Fatalf("owed %d, owing nothing %v; want every cancelled crawl drained once",
			f.amount(t), f.ledger.owingNothing)
	}
}

func TestACancelDropsVisitsWhileEveryPageVisitIsPaused(t *testing.T) {
	f := newFrontier(t)
	f.owe(t, pageVisitTo(t, "http://host/a"))
	if err := f.visits.Pause(t.Context()); err != nil {
		t.Fatalf("Pause: %v", err)
	}

	f.change(t, f.visits.Cancel, firstCrawl)
	f.drainCancelled(t)

	if f.amount(t) != 0 {
		t.Fatalf("owed %d, want the cancelled crawl drained", f.amount(t))
	}
}

func TestCancellingAHeldCrawlDropsItsVisits(t *testing.T) {
	f := newFrontier(t)
	f.owe(t, pageVisitTo(t, "http://host/a"))
	f.change(t, f.visits.Hold, firstCrawl)

	f.change(t, f.visits.Cancel, firstCrawl)
	f.change(t, f.visits.Release, firstCrawl)
	f.requireNoDueVisit(t)
	f.drainCancelled(t)

	if f.amount(t) != 0 || len(f.ledger.owingNothing) != 1 {
		t.Fatalf("owed %d, owing nothing %v; want the held crawl drained",
			f.amount(t), f.ledger.owingNothing)
	}
}

func TestAClosedCrawlTakesNoVisitAndIsNotDrained(t *testing.T) {
	f := newFrontier(t)

	f.change(t, f.visits.Close, firstCrawl)
	f.change(t, f.visits.Hold, firstCrawl)
	f.change(t, f.visits.Cancel, firstCrawl)
	f.owe(t, pageVisitTo(t, "http://host/a"))
	f.requireNoDueVisit(t)
	f.drainCancelled(t)

	if f.amount(t) != 0 || len(f.ledger.owingNothing) != 0 || len(f.ledger.cancelledOfHost) != 0 {
		t.Fatalf("owed %d, owing nothing %v, cancelled %v; want nothing",
			f.amount(t), f.ledger.owingNothing, f.ledger.cancelledOfHost)
	}
}

func TestAForgottenCrawlKeepsNothingAndItsVisitsAreOwedAgain(t *testing.T) {
	f := newFrontier(t)
	settled, owed := pageVisitTo(t, "http://host/a"), pageVisitTo(t, "http://host/b")
	f.owe(t, settled, owed)
	f.settleNext(t)
	f.change(t, f.visits.Close, firstCrawl)

	f.change(t, f.visits.ForgetCrawl, firstCrawl)

	if f.amount(t) != 0 {
		t.Fatalf("owed %d after forgetting, want none", f.amount(t))
	}
	f.owe(t, settled, owed)
	if f.amount(t) != 2 {
		t.Fatalf("owed %d, want both visits owed again", f.amount(t))
	}
	f.settleNext(t)
}

func TestAVisitWhoseCrawlCannotSettleIsStillAcknowledged(t *testing.T) {
	f := newFrontier(t)
	f.owe(t, pageVisitTo(t, "http://host/a"))
	f.ledger.refusing = errEngineRefusing

	f.settleNext(t)

	if f.amount(t) != 0 || !slices.Equal(f.failures.notSettled, []crawlid.CrawlID{firstCrawl}) ||
		len(f.failures.acknowledgments) != 0 {
		t.Fatalf("owed %d, not settled %v, ack failures %v; want the visit settled anyway",
			f.amount(t), f.failures.notSettled, f.failures.acknowledgments)
	}
}

func TestACancelTheLedgerRefusesIsReportedAndTriedAgain(t *testing.T) {
	f := newFrontier(t)
	f.owe(t, pageVisitTo(t, "http://host/a"))
	f.change(t, f.visits.Cancel, firstCrawl)
	f.ledger.refusing = errEngineRefusing

	f.drainCancelled(t)
	f.ledger.refusing = nil
	f.drainCancelled(t)

	if len(f.failures.drains) != 1 || f.amount(t) != 0 {
		t.Fatalf("drain failures %v, owed %d; want one failure, then drained",
			f.failures.drains, f.amount(t))
	}
}

func TestThePausedFrontierLendsNoVisitUntilResumed(t *testing.T) {
	f := newFrontier(t)
	f.owe(t, pageVisitTo(t, "http://host/a"))

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
	f.owe(t, pageVisitTo(t, "http://host/a"))
	message := f.next(t)
	f.engine.refusing = true

	message.Acknowledge(t.Context())
	message.ReturnAfter(t.Context(), time.Second)
	_, counted := f.visits.AmountOfPendingPageVisits(t.Context())
	f.requireNoDueVisit(t)
	f.drainCancelled(t)
	pauseErr := f.visits.Pause(t.Context())
	resumeErr := f.visits.Resume(t.Context())
	_, pausedErr := f.visits.IsPaused(t.Context())

	if len(f.failures.acknowledgments) != 1 || len(f.failures.returns) != 1 || counted ||
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

func TestConcurrentLeasesTakeEachOwedPageVisitOnce(t *testing.T) {
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
	f.owe(t, pageVisits...)

	taken := make(chan string, contendedVisits)
	var lessees sync.WaitGroup
	for range concurrentLessees {
		lessees.Go(func() {
			for {
				message, err := f.visits.Next(t.Context())
				if err != nil {
					return
				}
				message.Acknowledge(t.Context())
				taken <- pageVisitOf(t, message).URL.String()
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
