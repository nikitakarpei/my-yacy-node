package pagevisitintake_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitallowance"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

const (
	orderID  = "o1"
	visitURL = "http://host/"
)

var crawlID = crawlid.CrawlID{1}

type fakeAllowances struct {
	hostSpent      bool
	hostPageLimits []int
	deferrals      int
	maxDeferrals   int
	retries        int
	maxRetries     int
}

func newAllowances() *fakeAllowances {
	return &fakeAllowances{hostSpent: true, maxDeferrals: 2, maxRetries: 2}
}

func (a *fakeAllowances) TakePageOfHost(
	_ context.Context, _ pendingpagevisit.PendingPageVisit, maxPages int,
) (pagevisitallowance.Allowance, error) {
	a.hostPageLimits = append(a.hostPageLimits, maxPages)
	if !a.hostSpent {
		return pagevisitallowance.Allowance{
			DropReason: yacycrawlcontract.DropReasonHostPagesExhausted,
		}, nil
	}
	return pagevisitallowance.Allowance{Granted: true}, nil
}

func (a *fakeAllowances) TakeDeferral(
	_ context.Context, _ pendingpagevisit.PendingPageVisit, deferFor time.Duration,
) (pagevisitallowance.Allowance, error) {
	if a.deferrals >= a.maxDeferrals {
		return pagevisitallowance.Allowance{
			DropReason: yacycrawlcontract.DropReasonDeferralsExhausted,
		}, nil
	}
	a.deferrals++
	return pagevisitallowance.Allowance{Granted: true, PauseFor: deferFor}, nil
}

func (a *fakeAllowances) TakeAttempt(
	_ context.Context, _ pendingpagevisit.PendingPageVisit,
) (pagevisitallowance.Allowance, error) {
	if a.retries >= a.maxRetries {
		return pagevisitallowance.Allowance{
			DropReason: yacycrawlcontract.DropReasonRetriesExhausted,
		}, nil
	}
	a.retries++
	return pagevisitallowance.Allowance{Granted: true, PauseFor: time.Second}, nil
}

type fakeCrawls struct {
	profile   yacycrawlcontract.CrawlProfile
	forgotten bool
	err       error
}

func (o *fakeCrawls) OrderOf(
	_ *vault.Txn, _ crawlid.CrawlID,
) (acceptedorder.AcceptedOrder, bool, error) {
	if o.err != nil || o.forgotten {
		return acceptedorder.AcceptedOrder{}, false, o.err
	}
	seed, err := canonicalurl.CanonicalURLOf(visitURL)
	if err != nil {
		return acceptedorder.AcceptedOrder{}, false, err
	}
	order, err := acceptedorder.AcceptedOrderFrom(yacycrawlcontract.CrawlOrder{
		OrderID: orderID, Profile: o.profile, SeedURLs: []canonicalurl.CanonicalURL{seed},
	})
	return order, err == nil, err
}

type fakePageVisitor struct {
	mu       sync.Mutex
	outcomes []pagevisit.PageVisitOutcome
	visited  []string
}

func (f *fakePageVisitor) VisitPage(
	_ context.Context,
	_ acceptedorder.AcceptedOrder,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
) pagevisit.PageVisitOutcome {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.visited = append(f.visited, pendingPageVisit.URL.String())
	if len(f.outcomes) == 0 {
		return pagevisit.PageVisitOutcome{
			Conclusion: pagevisit.PageVisitCompleted, Result: yacycrawlcontract.ResultPublished,
		}
	}
	outcome := f.outcomes[0]
	f.outcomes = f.outcomes[1:]
	return outcome
}

func (f *fakePageVisitor) pageVisitCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.visited)
}

type recordingObserver struct {
	mu        sync.Mutex
	dropped   map[yacycrawlcontract.PageVisitDropReason]int
	completed map[yacycrawlcontract.PageVisitResult]int
	deferrals int
	retries   int
	returned  int
}

func newObserver() *recordingObserver {
	return &recordingObserver{
		dropped:   map[yacycrawlcontract.PageVisitDropReason]int{},
		completed: map[yacycrawlcontract.PageVisitResult]int{},
	}
}

func (o *recordingObserver) PendingPageVisitDropped(
	_ context.Context,
	_ pendingpagevisit.PendingPageVisit,
	reason yacycrawlcontract.PageVisitDropReason,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.dropped[reason]++
}

func (o *recordingObserver) PendingPageVisitDeferred(
	context.Context,
	pendingpagevisit.PendingPageVisit,
	time.Duration,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.deferrals++
}

func (o *recordingObserver) PendingPageVisitRetryScheduled(
	context.Context,
	pendingpagevisit.PendingPageVisit,
	time.Duration,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.retries++
}

func (o *recordingObserver) PendingPageVisitReturned(
	context.Context,
	pendingpagevisit.PendingPageVisit,
	error,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.returned++
}

func (o *recordingObserver) PendingPageVisitCompleted(
	_ context.Context,
	_ pendingpagevisit.PendingPageVisit,
	result yacycrawlcontract.PageVisitResult,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.completed[result]++
}

func wideProfile() yacycrawlcontract.CrawlProfile {
	return yacycrawlcontract.CrawlProfile{
		Scope:           yacycrawlcontract.ScopeWide,
		URLMustMatch:    yacycrawlcontract.MatchAll,
		MaxDepth:        5,
		MaxPagesPerHost: yacycrawlcontract.UnlimitedPagesPerHost,
	}
}

func leasedVisit(t *testing.T) pendingpagevisit.LeasedPageVisit {
	t.Helper()
	return pendingpagevisit.LeasedPageVisit{
		PageVisit: pendingpagevisit.PendingPageVisit{
			CrawlID: crawlID,
			URL:     canonicalurltest.CanonicalURLOf(t, visitURL),
		},
	}
}

type frontierCall string

const (
	ended    frontierCall = "ended"
	returned frontierCall = "returned"
)

type fakeFrontier struct {
	leased        []pendingpagevisit.LeasedPageVisit
	endIntake     context.CancelFunc
	calls         []frontierCall
	returnedAfter time.Duration
}

func (f *fakeFrontier) Next(context.Context) (pendingpagevisit.LeasedPageVisit, error) {
	if len(f.leased) == 0 {
		f.endIntake()
		return pendingpagevisit.LeasedPageVisit{}, context.Canceled
	}
	next := f.leased[0]
	f.leased = f.leased[1:]
	return next, nil
}

func (f *fakeFrontier) End(context.Context, pendingpagevisit.LeasedPageVisit) {
	f.calls = append(f.calls, ended)
}

func (f *fakeFrontier) Return(
	_ context.Context,
	_ pendingpagevisit.LeasedPageVisit,
	delay time.Duration,
) {
	f.calls = append(f.calls, returned)
	f.returnedAfter = delay
}

type everyVisitAdmitted struct{}

func (everyVisitAdmitted) AwaitAdmission(context.Context) error {
	return nil
}

type noVisitAdmitted struct{}

func (noVisitAdmitted) AwaitAdmission(context.Context) error {
	return errors.New("the crawled pages are not read")
}

type crawlWorker struct {
	frontier    *fakeFrontier
	allowances  *fakeAllowances
	orders      *fakeCrawls
	pageVisitor *fakePageVisitor
	observer    *recordingObserver
}

func newWorker() *crawlWorker {
	return &crawlWorker{
		frontier:    &fakeFrontier{},
		allowances:  newAllowances(),
		orders:      &fakeCrawls{profile: wideProfile()},
		pageVisitor: &fakePageVisitor{},
		observer:    newObserver(),
	}
}

func (w *crawlWorker) consume(t *testing.T, visits ...pendingpagevisit.LeasedPageVisit) error {
	t.Helper()
	return w.consumeAdmittedBy(t, everyVisitAdmitted{}, visits...)
}

func (w *crawlWorker) consumeAdmittedBy(
	t *testing.T,
	admission pullintake.Admission,
	visits ...pendingpagevisit.LeasedPageVisit,
) error {
	t.Helper()
	storage, err := vault.New(memoryvault.OpenEngine(0), nil, nil, vaultenginetest.FrozenClock{})
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	ctx, endIntake := context.WithCancel(t.Context())
	defer endIntake()
	w.frontier.leased = visits
	w.frontier.endIntake = endIntake
	return pagevisitintake.NewPageVisitConsumer(
		storage,
		w.frontier,
		w.allowances,
		w.orders,
		w.pageVisitor,
		w.observer,
		admission,
		1,
	).Run(ctx)
}

func TestAPageVisitIsVisitedThenAcknowledged(t *testing.T) {
	worker := newWorker()
	visit := leasedVisit(t)

	if err := worker.consume(t, visit); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.pageVisitor.pageVisitCount() != 1 {
		t.Fatalf("visited %d urls, want 1", worker.pageVisitor.pageVisitCount())
	}
	if got := worker.frontier.calls; len(got) != 1 || got[0] != ended {
		t.Fatalf("frontier calls %v, want the visit ended once", got)
	}
}

func TestADeferredVisitReturnsAfterTheDelayTheSiteAsked(t *testing.T) {
	worker := newWorker()
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{{
		Conclusion: pagevisit.PageVisitDeferred, DeferFor: 7 * time.Second,
	}}
	visit := leasedVisit(t)

	if err := worker.consume(t, visit); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := worker.frontier.calls; len(got) != 1 || got[0] != returned {
		t.Fatalf("frontier calls %v, want the visit returned once", got)
	}
	if worker.frontier.returnedAfter != 7*time.Second {
		t.Fatalf("held back for %v, want the delay the site asked", worker.frontier.returnedAfter)
	}
	if worker.observer.deferrals != 1 {
		t.Fatalf("observer honored %d deferrals, want one", worker.observer.deferrals)
	}
}

func TestAURLThatExhaustedItsDeferralsIsDropped(t *testing.T) {
	worker := newWorker()
	worker.allowances.deferrals = 2
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{
		{Conclusion: pagevisit.PageVisitDeferred},
	}
	visit := leasedVisit(t)

	if err := worker.consume(t, visit); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.observer.dropped[yacycrawlcontract.DropReasonDeferralsExhausted] != 1 {
		t.Fatalf("observer dropped %v, want deferrals exhausted", worker.observer.dropped)
	}
	if got := worker.frontier.calls; len(got) != 1 || got[0] != ended {
		t.Fatalf("frontier calls %v, want the visit ended once", got)
	}
}

func TestARetryableVisitReturnsAfterItsBackoff(t *testing.T) {
	worker := newWorker()
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{
		{Conclusion: pagevisit.PageVisitRetryable},
	}
	visit := leasedVisit(t)

	if err := worker.consume(t, visit); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.frontier.returnedAfter != time.Second {
		t.Fatalf("held back for %v, want the first backoff", worker.frontier.returnedAfter)
	}
}

func TestAReturnedVisitComesBackWithoutSpendingAnAttempt(t *testing.T) {
	worker := newWorker()
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{{
		Conclusion:  pagevisit.PageVisitReturned,
		ReturnCause: errors.New("stream down"),
	}}
	visit := leasedVisit(t)

	if err := worker.consume(t, visit); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := worker.frontier.calls; len(got) != 1 || got[0] != returned {
		t.Fatalf("frontier calls %v, want the visit returned once", got)
	}
	if worker.allowances.retries != 0 {
		t.Fatalf("attempts spent = %d, want none", worker.allowances.retries)
	}
}

func TestAVisitTheAdmissionHoldsBackIsNeitherVisitedNorEnded(t *testing.T) {
	worker := newWorker()
	visit := leasedVisit(t)

	if err := worker.consumeAdmittedBy(t, noVisitAdmitted{}, visit); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.pageVisitor.pageVisitCount() != 0 {
		t.Fatal("a held back visit was visited")
	}
	if got := worker.frontier.calls; len(got) != 0 {
		t.Fatalf("frontier calls %v, want the visit untouched", got)
	}
}

func TestAURLThatExhaustedItsRetriesIsDropped(t *testing.T) {
	worker := newWorker()
	worker.allowances.retries = 2
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{
		{Conclusion: pagevisit.PageVisitRetryable},
	}

	if err := worker.consume(t, leasedVisit(t)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.observer.dropped[yacycrawlcontract.DropReasonRetriesExhausted] != 1 {
		t.Fatalf("observer dropped %v, want retries exhausted", worker.observer.dropped)
	}
}

func TestAURLWhoseHostExhaustedItsPagesIsDroppedBeforeTheFetch(t *testing.T) {
	worker := newWorker()
	worker.allowances.hostSpent = false
	visit := leasedVisit(t)

	if err := worker.consume(t, visit); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.pageVisitor.pageVisitCount() != 0 {
		t.Fatal("a host that spent its pages should not be fetched again")
	}
	if worker.observer.dropped[yacycrawlcontract.DropReasonHostPagesExhausted] != 1 {
		t.Fatalf("observer dropped %v, want host pages exhausted", worker.observer.dropped)
	}
}

func TestTheProfilesHostPageLimitReachesTheAllowances(t *testing.T) {
	worker := newWorker()
	worker.orders.profile.MaxPagesPerHost = 7

	if err := worker.consume(t, leasedVisit(t)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(worker.allowances.hostPageLimits) != 1 || worker.allowances.hostPageLimits[0] != 7 {
		t.Fatalf(
			"the host page was taken against %v, want the profile limit of 7",
			worker.allowances.hostPageLimits,
		)
	}
}

func TestAVisitLeasedAgainAsksForItsHostPageAgain(t *testing.T) {
	worker := newWorker()
	visit := leasedVisit(t)

	if err := worker.consume(t, visit, visit); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(worker.allowances.hostPageLimits) != 2 {
		t.Fatalf(
			"asked for %d host pages, want one for each delivery",
			len(worker.allowances.hostPageLimits),
		)
	}
}

func TestAVisitOfAnUnreadableOrderReturnsForRedelivery(t *testing.T) {
	worker := newWorker()
	worker.orders.err = errors.New("bucket down")
	visit := leasedVisit(t)

	if err := worker.consume(t, visit); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := worker.frontier.calls; len(got) != 1 || got[0] != returned {
		t.Fatalf("frontier calls %v, want the visit returned once", got)
	}
}

func TestAVisitWhoseCrawlIsForgottenReturnsUnvisited(t *testing.T) {
	worker := newWorker()
	worker.orders.forgotten = true
	visit := leasedVisit(t)

	if err := worker.consume(t, visit); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := worker.frontier.calls; len(got) != 1 || got[0] != returned {
		t.Fatalf("frontier calls %v, want the visit returned once", got)
	}
	if worker.pageVisitor.pageVisitCount() != 0 {
		t.Fatal("a url of a forgotten order should not be visited")
	}
}

func TestAVisitWhoseOrderProfileIsUnreadableReturnsForRedelivery(t *testing.T) {
	worker := newWorker()
	worker.orders.profile.URLMustNotMatch = "([unclosed"
	visit := leasedVisit(t)

	if err := worker.consume(t, visit); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := worker.frontier.calls; len(got) != 1 || got[0] != returned {
		t.Fatalf("frontier calls %v, want the visit returned once", got)
	}
	if worker.pageVisitor.pageVisitCount() != 0 {
		t.Fatal("a url under an unreadable profile should not be visited")
	}
}

func TestTheResultOfACompletedVisitIsObserved(t *testing.T) {
	worker := newWorker()
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{{
		Conclusion: pagevisit.PageVisitCompleted, Result: yacycrawlcontract.ResultFetchRejected,
	}}

	if err := worker.consume(t, leasedVisit(t)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.observer.completed[yacycrawlcontract.ResultFetchRejected] != 1 {
		t.Fatalf("observer completed %v, want fetch rejected", worker.observer.completed)
	}
}

func TestAVisitTheVisitorDropsIsObservedAsDropped(t *testing.T) {
	worker := newWorker()
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{{
		Conclusion: pagevisit.PageVisitDropped, DropReason: yacycrawlcontract.DropReasonNotDue,
	}}

	if err := worker.consume(t, leasedVisit(t)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.observer.dropped[yacycrawlcontract.DropReasonNotDue] != 1 ||
		len(worker.observer.completed) != 0 {
		t.Fatalf("observer dropped %v and completed %v, want only not due dropped",
			worker.observer.dropped, worker.observer.completed)
	}
}
