package pagevisitintake_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitallowance"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitdrop"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

const (
	orderID  = "o1"
	visitURL = "http://host/"
)

var crawlID = crawlid.CrawlID{1}

type fakeTakenPageVisits struct {
	takers  map[string]string
	takeErr error
}

func newTakenPageVisits() *fakeTakenPageVisits {
	return &fakeTakenPageVisits{takers: map[string]string{}}
}

func (visits *fakeTakenPageVisits) TakePageVisit(
	_ *vault.Txn, _ crawlid.CrawlID, url canonicalurl.CanonicalURL, taker string,
) (bool, error) {
	if visits.takeErr != nil {
		return false, visits.takeErr
	}
	standing, taken := visits.takers[url.String()]
	if !taken {
		visits.takers[url.String()] = taker
		return true, nil
	}
	return standing == taker, nil
}

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
		return pagevisitallowance.Allowance{DropReason: pagevisitdrop.HostPagesExhausted}, nil
	}
	return pagevisitallowance.Allowance{Granted: true}, nil
}

func (a *fakeAllowances) TakeDeferral(
	_ context.Context, _ pendingpagevisit.PendingPageVisit, deferFor time.Duration,
) (pagevisitallowance.Allowance, error) {
	if a.deferrals >= a.maxDeferrals {
		return pagevisitallowance.Allowance{DropReason: pagevisitdrop.DeferralsExhausted}, nil
	}
	a.deferrals++
	return pagevisitallowance.Allowance{Granted: true, PauseFor: deferFor}, nil
}

func (a *fakeAllowances) TakeAttempt(
	_ context.Context, _ pendingpagevisit.PendingPageVisit,
) (pagevisitallowance.Allowance, error) {
	if a.retries >= a.maxRetries {
		return pagevisitallowance.Allowance{DropReason: pagevisitdrop.RetriesExhausted}, nil
	}
	a.retries++
	return pagevisitallowance.Allowance{Granted: true, PauseFor: time.Second}, nil
}

type fakeCrawls struct {
	profile   yacycrawlcontract.CrawlProfile
	seeds     []canonicalurl.CanonicalURL
	forgotten bool
	err       error
}

func (o *fakeCrawls) OrderOf(
	_ *vault.Txn, _ crawlid.CrawlID,
) (acceptedorder.AcceptedOrder, bool, error) {
	if o.err != nil || o.forgotten {
		return acceptedorder.AcceptedOrder{}, false, o.err
	}
	order, err := acceptedorder.AcceptedOrderFrom(yacycrawlcontract.CrawlOrder{
		OrderID: orderID, Profile: o.profile, SeedURLs: o.seeds,
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
			Conclusion: pagevisit.PageVisitCompleted, Result: pagevisitresult.Published,
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
	dropped   map[pagevisitdrop.Reason]int
	completed map[pagevisitresult.Result]int
	deferrals int
	retries   int
	returned  int
}

func newObserver() *recordingObserver {
	return &recordingObserver{
		dropped:   map[pagevisitdrop.Reason]int{},
		completed: map[pagevisitresult.Result]int{},
	}
}

func (o *recordingObserver) PendingPageVisitDropped(
	_ context.Context,
	_ pendingpagevisit.PendingPageVisit,
	reason pagevisitdrop.Reason,
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
	result pagevisitresult.Result,
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

func visitMessage(t *testing.T, sequence uint64) *pullintaketest.Message {
	t.Helper()
	data, err := pendingpagevisit.MarshalPendingPageVisit(pendingpagevisit.PendingPageVisit{
		CrawlID: crawlID,
		URL:     canonicalurltest.CanonicalURLOf(t, visitURL),
		Depth:   0,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &pullintaketest.Message{Body: data, Sequence: sequence}
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
	pageVisits  *fakeTakenPageVisits
	allowances  *fakeAllowances
	orders      *fakeCrawls
	pageVisitor *fakePageVisitor
	observer    *recordingObserver
}

func newWorker() *crawlWorker {
	return &crawlWorker{
		pageVisits:  newTakenPageVisits(),
		allowances:  newAllowances(),
		orders:      &fakeCrawls{profile: wideProfile()},
		pageVisitor: &fakePageVisitor{},
		observer:    newObserver(),
	}
}

func (w *crawlWorker) consume(t *testing.T, messages ...jetstream.Msg) error {
	t.Helper()
	return w.consumeAdmittedBy(t, everyVisitAdmitted{}, messages...)
}

func (w *crawlWorker) consumeAdmittedBy(
	t *testing.T,
	admission pullintake.Admission,
	messages ...jetstream.Msg,
) error {
	t.Helper()
	storage, err := vault.New(memoryvault.OpenEngine(0), nil, nil, vaultenginetest.FrozenClock{})
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	return pagevisitintake.NewPageVisitConsumer(
		storage,
		pullintake.PendingMessagesOf(pullintaketest.MessageSourceOf(messages...)),
		w.pageVisits,
		w.allowances,
		w.orders,
		w.pageVisitor,
		w.observer,
		admission,
		1,
	).Run(context.Background())
}

func TestAClaimedURLIsVisitedThenAcknowledged(t *testing.T) {
	worker := newWorker()
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.pageVisitor.pageVisitCount() != 1 {
		t.Fatalf("visited %d urls, want 1", worker.pageVisitor.pageVisitCount())
	}
	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.Acknowledged {
		t.Fatalf("message settled %v, want one ack", got)
	}
}

func TestASecondFrontierMessageForAClaimedURLIsDropped(t *testing.T) {
	worker := newWorker()
	duplicate := visitMessage(t, 2)

	if err := worker.consume(t, visitMessage(t, 1), duplicate); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.pageVisitor.pageVisitCount() != 1 {
		t.Fatalf("visited %d times, want the url fetched once", worker.pageVisitor.pageVisitCount())
	}
	if got := duplicate.Settlements(); len(got) != 1 || got[0] != pullintaketest.Acknowledged {
		t.Fatalf("duplicate settled %v, want one ack", got)
	}
}

func TestARedeliveredMessageResumesItsOwnClaim(t *testing.T) {
	worker := newWorker()
	message := visitMessage(t, 1)

	if err := worker.consume(t, message, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.pageVisitor.pageVisitCount() != 2 {
		t.Fatal("a redelivery should visit the claim it left behind")
	}
}

func TestADeferredVisitReturnsAfterTheDelayTheSiteAsked(t *testing.T) {
	worker := newWorker()
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{{
		Conclusion: pagevisit.PageVisitDeferred, DeferFor: 7 * time.Second,
	}}
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.HeldBack {
		t.Fatalf("message settled %v, want one delayed return", got)
	}
	if message.HeldBackFor() != 7*time.Second {
		t.Fatalf("held back for %v, want the delay the site asked", message.HeldBackFor())
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
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.observer.dropped[pagevisitdrop.DeferralsExhausted] != 1 {
		t.Fatalf("observer dropped %v, want deferrals exhausted", worker.observer.dropped)
	}
	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.Acknowledged {
		t.Fatalf("message settled %v, want one ack", got)
	}
}

func TestARetryableVisitReturnsAfterItsBackoff(t *testing.T) {
	worker := newWorker()
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{
		{Conclusion: pagevisit.PageVisitRetryable},
	}
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if message.HeldBackFor() != time.Second {
		t.Fatalf("held back for %v, want the first backoff", message.HeldBackFor())
	}
}

func TestAReturnedVisitComesBackWithoutSpendingAnAttempt(t *testing.T) {
	worker := newWorker()
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{{
		Conclusion:  pagevisit.PageVisitReturned,
		ReturnCause: errors.New("stream down"),
	}}
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.HeldBack {
		t.Fatalf("message settled %v, want one delayed return", got)
	}
	if worker.allowances.retries != 0 {
		t.Fatalf("attempts spent = %d, want none", worker.allowances.retries)
	}
}

func TestAVisitTheAdmissionHoldsBackIsNeitherVisitedNorSettled(t *testing.T) {
	worker := newWorker()
	message := visitMessage(t, 1)

	if err := worker.consumeAdmittedBy(t, noVisitAdmitted{}, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.pageVisitor.pageVisitCount() != 0 {
		t.Fatal("a held back visit was visited")
	}
	if got := message.Settlements(); len(got) != 0 {
		t.Fatalf("message settled %v, want it untouched", got)
	}
}

func TestAURLThatExhaustedItsRetriesIsDropped(t *testing.T) {
	worker := newWorker()
	worker.allowances.retries = 2
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{
		{Conclusion: pagevisit.PageVisitRetryable},
	}

	if err := worker.consume(t, visitMessage(t, 1)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.observer.dropped[pagevisitdrop.RetriesExhausted] != 1 {
		t.Fatalf("observer dropped %v, want retries exhausted", worker.observer.dropped)
	}
}

func TestAURLWhoseHostExhaustedItsPagesIsDroppedBeforeTheFetch(t *testing.T) {
	worker := newWorker()
	worker.allowances.hostSpent = false
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.pageVisitor.pageVisitCount() != 0 {
		t.Fatal("a host that spent its pages should not be fetched again")
	}
	if worker.observer.dropped[pagevisitdrop.HostPagesExhausted] != 1 {
		t.Fatalf("observer dropped %v, want host pages exhausted", worker.observer.dropped)
	}
}

func TestTheProfilesHostPageLimitReachesTheAllowances(t *testing.T) {
	worker := newWorker()
	worker.orders.profile.MaxPagesPerHost = 7

	if err := worker.consume(t, visitMessage(t, 1)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(worker.allowances.hostPageLimits) != 1 || worker.allowances.hostPageLimits[0] != 7 {
		t.Fatalf(
			"the host page was taken against %v, want the profile limit of 7",
			worker.allowances.hostPageLimits,
		)
	}
}

func TestARedeliveredMessageAsksForItsHostPageAgain(t *testing.T) {
	worker := newWorker()
	message := visitMessage(t, 1)

	if err := worker.consume(t, message, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(worker.allowances.hostPageLimits) != 2 {
		t.Fatalf(
			"asked for %d host pages, want one for each delivery",
			len(worker.allowances.hostPageLimits),
		)
	}
}

func TestAPageVisitNoOneCanTakeReturnsForRedelivery(t *testing.T) {
	worker := newWorker()
	worker.pageVisits.takeErr = errors.New("bucket down")
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.pageVisitor.pageVisitCount() != 0 {
		t.Fatal("a url with no answered claim should not be visited")
	}
	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.HeldBack {
		t.Fatalf("message settled %v, want one delayed return", got)
	}
}

func TestAVisitOfAnUnreadableOrderReturnsForRedelivery(t *testing.T) {
	worker := newWorker()
	worker.orders.err = errors.New("bucket down")
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.HeldBack {
		t.Fatalf("message settled %v, want one delayed return", got)
	}
}

func TestAVisitWhoseOrderIsForgottenIsDroppedUnvisited(t *testing.T) {
	worker := newWorker()
	worker.orders.forgotten = true
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.Acknowledged {
		t.Fatalf("message settled %v, want one ack", got)
	}
	if worker.observer.dropped[pagevisitdrop.OrderForgotten] != 1 {
		t.Fatalf("dropped %v, want one forgotten order", worker.observer.dropped)
	}
	if worker.pageVisitor.pageVisitCount() != 0 {
		t.Fatal("a url of a forgotten order should not be visited")
	}
}

func TestAVisitWhoseOrderProfileIsUnreadableReturnsForRedelivery(t *testing.T) {
	worker := newWorker()
	worker.orders.profile.URLMustNotMatch = "([unclosed"
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.HeldBack {
		t.Fatalf("message settled %v, want one delayed return", got)
	}
	if worker.pageVisitor.pageVisitCount() != 0 {
		t.Fatal("a url under an unreadable profile should not be visited")
	}
}

func TestAnUndecodablePendingVisitHaltsIntake(t *testing.T) {
	if err := newWorker().consume(t, &pullintaketest.Message{Body: []byte("{"), Sequence: 1}); err == nil {
		t.Fatal("an undecodable pending page visit should halt intake")
	}
}

func TestTheResultOfACompletedVisitIsObserved(t *testing.T) {
	worker := newWorker()
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{{
		Conclusion: pagevisit.PageVisitCompleted, Result: pagevisitresult.FetchRejected,
	}}

	if err := worker.consume(t, visitMessage(t, 1)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.observer.completed[pagevisitresult.FetchRejected] != 1 {
		t.Fatalf("observer completed %v, want fetch rejected", worker.observer.completed)
	}
}

func TestAVisitTheVisitorDropsIsObservedAsDropped(t *testing.T) {
	worker := newWorker()
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{{
		Conclusion: pagevisit.PageVisitDropped, DropReason: pagevisitdrop.NotDue,
	}}

	if err := worker.consume(t, visitMessage(t, 1)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.observer.dropped[pagevisitdrop.NotDue] != 1 || len(worker.observer.completed) != 0 {
		t.Fatalf("observer dropped %v and completed %v, want only not due dropped",
			worker.observer.dropped, worker.observer.completed)
	}
}
