package frontier_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
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
	lease              = time.Minute
	concurrentLessees  = 8
	contendedVisits    = 32
	visitsOverOneBatch = 300
)

var (
	errNoDueVisit     = errors.New("no page visit is due")
	errEngineRefusing = errors.New("engine refuses")
	crawlID           = crawlid.CrawlID{1}
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

type recordedFailures struct {
	guard           sync.Mutex
	leases          []error
	counts          []error
	acknowledgments []string
	returns         []string
}

func (r *recordedFailures) FrontierLeaseFailed(_ context.Context, err error) {
	r.guard.Lock()
	defer r.guard.Unlock()
	r.leases = append(r.leases, err)
}

func (r *recordedFailures) FrontierCountFailed(_ context.Context, err error) {
	r.counts = append(r.counts, err)
}

func (r *recordedFailures) FrontierAcknowledgeFailed(
	_ context.Context,
	identity string,
	_ error,
) {
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

type refusingEngine struct {
	vault.Engine
	refusing        bool
	refusingRecords bool
}

func (engine *refusingEngine) Update(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if engine.refusing {
		return errEngineRefusing
	}
	return engine.Engine.Update(ctx, func(etx vault.EngineTxn) error {
		if engine.refusingRecords {
			return fn(refusingTxn{EngineTxn: etx})
		}
		return fn(etx)
	})
}

type refusingTxn struct {
	vault.EngineTxn
}

func (txn refusingTxn) Records(name vault.Name) vault.EngineRecords {
	return refusingRecords{EngineRecords: txn.EngineTxn.Records(name)}
}

type refusingRecords struct {
	vault.EngineRecords
}

func (refusingRecords) Get([]byte) ([]byte, error) {
	return nil, errEngineRefusing
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
	failures := &recordedFailures{}
	visits, err := frontier.Open(t.Context(), storage, lease, visitClock, failures)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return frontierScene{
		storage: storage, engine: refusing, clock: visitClock, failures: failures, visits: visits,
	}
}

func pageVisitTo(t *testing.T, rawURL string) pendingpagevisit.PendingPageVisit {
	t.Helper()
	return pendingpagevisit.PendingPageVisit{
		CrawlID: crawlID,
		URL:     canonicalurltest.CanonicalURLOf(t, rawURL),
		Depth:   1,
	}
}

func (f frontierScene) publish(t *testing.T, pageVisits ...pendingpagevisit.PendingPageVisit) {
	t.Helper()
	if err := f.owe(t, pageVisits); err != nil {
		t.Fatalf("Owe: %v", err)
	}
}

func (f frontierScene) owe(t *testing.T, pageVisits []pendingpagevisit.PendingPageVisit) error {
	t.Helper()
	return f.storage.Update(t.Context(), func(tx *vault.Txn) error {
		return f.visits.Owe(tx, pageVisits)
	})
}

func (f frontierScene) next(t *testing.T) pullintake.PendingMessage {
	t.Helper()
	message, err := f.visits.Next(t.Context())
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	return message
}

func (f frontierScene) requireNoDueVisit(t *testing.T) {
	t.Helper()
	if message, err := f.visits.Next(t.Context()); !errors.Is(err, errNoDueVisit) {
		t.Fatalf("Next = %v, %v; want no due page visit", message, err)
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

func TestAPublishedPageVisitIsTakenWithItsArrival(t *testing.T) {
	f := newFrontier(t)
	pageVisit := pageVisitTo(t, "http://host/page")
	f.publish(t, pageVisit)
	arrival := f.clock.Now()

	message := f.next(t)

	if got := pageVisitOf(t, message); got != pageVisit {
		t.Fatalf("page visit = %+v, want %+v", got, pageVisit)
	}
	if !message.Arrival().Equal(arrival) {
		t.Fatalf("Arrival = %v, want %v", message.Arrival(), arrival)
	}
}

func TestAPageVisitPublishedTwiceWhilePendingIsTakenOnce(t *testing.T) {
	f := newFrontier(t)
	pageVisit := pageVisitTo(t, "http://host/page")
	f.publish(t, pageVisit, pageVisit)
	f.publish(t, pageVisit)

	f.next(t)

	f.requireNoDueVisit(t)
}

func TestEveryPageVisitOfOnePublicationIsPending(t *testing.T) {
	f := newFrontier(t)
	pageVisits := make([]pendingpagevisit.PendingPageVisit, 0, visitsOverOneBatch)
	for page := range visitsOverOneBatch {
		pageVisits = append(pageVisits, pageVisitTo(t, fmt.Sprintf("http://host/page%d", page)))
	}

	f.publish(t, pageVisits...)

	if amount := f.amount(t); amount != visitsOverOneBatch {
		t.Fatalf("pending page visits = %d, want %d", amount, visitsOverOneBatch)
	}
}

func TestATakenPageVisitComesBackOnlyOnceItsLeaseEnds(t *testing.T) {
	f := newFrontier(t)
	f.publish(t, pageVisitTo(t, "http://host/page"))
	taken := f.next(t)

	f.clock.pass(lease - time.Millisecond)
	f.requireNoDueVisit(t)
	f.clock.pass(time.Millisecond)

	if again := f.next(t); again.Identity() != taken.Identity() {
		t.Fatalf("Identity = %q after the lease, want %q", again.Identity(), taken.Identity())
	}
}

func TestAnAcknowledgedPageVisitNeverComesBack(t *testing.T) {
	f := newFrontier(t)
	f.publish(t, pageVisitTo(t, "http://host/page"))

	f.next(t).Acknowledge(t.Context())

	f.clock.pass(lease)
	f.requireNoDueVisit(t)
}

func TestAPageVisitPublishedAgainAfterItsAcknowledgmentCarriesANewIdentity(t *testing.T) {
	f := newFrontier(t)
	pageVisit := pageVisitTo(t, "http://host/page")
	f.publish(t, pageVisit)
	first := f.next(t)
	first.Acknowledge(t.Context())

	f.publish(t, pageVisit)

	if again := f.next(t); again.Identity() == first.Identity() {
		t.Fatalf("Identity = %q again, want a new identity", again.Identity())
	}
}

func TestAPageVisitReturnedAfterADelayIsDueOnlyOnceTheDelayPasses(t *testing.T) {
	f := newFrontier(t)
	f.publish(t, pageVisitTo(t, "http://host/page"))
	delay := 3 * lease

	f.next(t).ReturnAfter(t.Context(), delay)

	f.clock.pass(delay - time.Millisecond)
	f.requireNoDueVisit(t)
	f.clock.pass(time.Millisecond)
	f.next(t)
}

func TestAReturnedPageVisitComesBackLater(t *testing.T) {
	f := newFrontier(t)
	f.publish(t, pageVisitTo(t, "http://host/page"))

	f.next(t).Return(t.Context())

	f.requireNoDueVisit(t)
	f.clock.pass(time.Hour)
	f.next(t)
}

func TestAReturnOfAnAcknowledgedPageVisitLeavesItGone(t *testing.T) {
	f := newFrontier(t)
	f.publish(t, pageVisitTo(t, "http://host/page"))
	message := f.next(t)
	message.Acknowledge(t.Context())

	message.ReturnAfter(t.Context(), time.Second)

	if amount := f.amount(t); amount != 0 {
		t.Fatalf("pending page visits = %d, want none", amount)
	}
}

func TestPageVisitsAreTakenInTheOrderTheyFallDue(t *testing.T) {
	f := newFrontier(t)
	first, second := pageVisitTo(t, "http://host/first"), pageVisitTo(t, "http://host/second")
	f.publish(t, first)
	f.clock.pass(time.Second)
	f.publish(t, second)

	f.next(t).ReturnAfter(t.Context(), time.Second)
	f.clock.pass(time.Second)

	if got := pageVisitOf(t, f.next(t)); got != second {
		t.Fatalf("page visit = %+v, want the one due first %+v", got, second)
	}
}

func TestAnOweTheStorageRefusesFails(t *testing.T) {
	f := newFrontier(t)
	f.engine.refusingRecords = true

	if err := f.owe(
		t,
		[]pendingpagevisit.PendingPageVisit{pageVisitTo(t, "http://host/page")},
	); !errors.Is(err, errEngineRefusing) {
		t.Fatalf("Owe err = %v, want the refusal", err)
	}
}

func TestALeaseTheStorageRefusesIsReportedAndNothingIsTaken(t *testing.T) {
	f := newFrontier(t)
	f.publish(t, pageVisitTo(t, "http://host/page"))
	f.engine.refusing = true

	f.requireNoDueVisit(t)

	if len(f.failures.leases) != 1 {
		t.Fatalf("lease failures = %v, want one", f.failures.leases)
	}
}

func TestSettlementsTheStorageRefusesAreReported(t *testing.T) {
	f := newFrontier(t)
	f.publish(t, pageVisitTo(t, "http://host/page"))
	message := f.next(t)
	f.engine.refusing = true

	message.Acknowledge(t.Context())
	message.ReturnAfter(t.Context(), time.Second)

	identity := message.Identity()
	if len(f.failures.acknowledgments) != 1 || f.failures.acknowledgments[0] != identity ||
		len(f.failures.returns) != 1 || f.failures.returns[0] != identity {
		t.Fatalf(
			"failures = %+v, want one acknowledgment and one return of %q",
			f.failures,
			identity,
		)
	}
}

func TestTheAmountOfPendingPageVisitsCountsTakenOnesUntilTheyAreAcknowledged(t *testing.T) {
	f := newFrontier(t)
	f.publish(t, pageVisitTo(t, "http://host/first"), pageVisitTo(t, "http://host/second"))
	f.next(t).Acknowledge(t.Context())
	f.next(t)

	if amount := f.amount(t); amount != 1 {
		t.Fatalf("AmountOfPendingPageVisits = %d; want 1", amount)
	}
}

func TestACountTheStorageRefusesIsReported(t *testing.T) {
	f := newFrontier(t)
	f.engine.refusing = true

	if _, counted := f.visits.AmountOfPendingPageVisits(t.Context()); counted {
		t.Fatal("AmountOfPendingPageVisits counted on a refusing storage")
	}
	if len(f.failures.counts) != 1 {
		t.Fatalf("count failures = %v, want one", f.failures.counts)
	}
}

func TestConcurrentLeasesTakeEachDuePageVisitOnce(t *testing.T) {
	engine, err := pebblevault.OpenEngine(
		filepath.Join(t.TempDir(), "crawl-state"), 0, pebblevault.MachineLimits{}, nil, nil,
	)
	if err != nil {
		t.Fatalf("OpenEngine: %v", err)
	}
	f := newFrontierOn(t, engine)
	published := make([]pendingpagevisit.PendingPageVisit, 0, contendedVisits)
	for page := range contendedVisits {
		published = append(published, pageVisitTo(t, fmt.Sprintf("http://host/page%d", page)))
	}
	f.publish(t, published...)

	leasedByLessee := make([][]string, concurrentLessees)
	var leasing sync.WaitGroup
	for lessee := range concurrentLessees {
		leasing.Go(func() {
			for {
				message, err := f.visits.Next(context.Background())
				if err != nil {
					return
				}
				leasedByLessee[lessee] = append(leasedByLessee[lessee], message.Identity())
			}
		})
	}
	leasing.Wait()

	leased := slices.Concat(leasedByLessee...)
	slices.Sort(leased)
	if len(leased) != contendedVisits || len(slices.Compact(leased)) != contendedVisits {
		t.Fatalf("leased %d page visits, want each of %d once", len(leased), contendedVisits)
	}
}
