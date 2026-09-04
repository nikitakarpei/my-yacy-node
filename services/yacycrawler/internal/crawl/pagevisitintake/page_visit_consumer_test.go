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
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitallowance"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitclaim"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

const (
	orderID  = "o1"
	visitURL = "http://host/"
)

type fakeClaims struct {
	holders  map[string]string
	claimErr error
}

func newClaims() *fakeClaims {
	return &fakeClaims{holders: map[string]string{}}
}

func (c *fakeClaims) Claim(
	_ context.Context, _ string, url canonicalurl.CanonicalURL, holder string,
) (pagevisitclaim.Claim, error) {
	if c.claimErr != nil {
		return pagevisitclaim.Unanswered, c.claimErr
	}
	standing, held := c.holders[url.String()]
	if !held {
		c.holders[url.String()] = holder
		return pagevisitclaim.Taken, nil
	}
	if standing != holder {
		return pagevisitclaim.HeldElsewhere, nil
	}
	return pagevisitclaim.Resumed, nil
}

type fakeLedger struct {
	hostSpent      bool
	hostPageLimits []int
	deferrals      int
	maxDeferrals   int
	retries        int
	maxRetries     int
}

func newLedger() *fakeLedger {
	return &fakeLedger{hostSpent: true, maxDeferrals: 2, maxRetries: 2}
}

func (a *fakeLedger) HostPageFor(
	_ context.Context, _ pendingpagevisit.PendingPageVisit, maxPages int,
) (pagevisitallowance.Allowance, error) {
	a.hostPageLimits = append(a.hostPageLimits, maxPages)
	if !a.hostSpent {
		return pagevisitallowance.Allowance{Exhausted: disposal.HostPagesExhausted}, nil
	}
	return pagevisitallowance.Allowance{Granted: true}, nil
}

func (a *fakeLedger) DeferralFor(
	_ context.Context, _ pendingpagevisit.PendingPageVisit, deferFor time.Duration,
) (pagevisitallowance.Allowance, error) {
	if a.deferrals >= a.maxDeferrals {
		return pagevisitallowance.Allowance{Exhausted: disposal.DeferralsExhausted}, nil
	}
	a.deferrals++
	return pagevisitallowance.Allowance{Granted: true, PauseFor: deferFor}, nil
}

func (a *fakeLedger) AnotherAttemptFor(
	_ context.Context, _ pendingpagevisit.PendingPageVisit,
) (pagevisitallowance.Allowance, error) {
	if a.retries >= a.maxRetries {
		return pagevisitallowance.Allowance{Exhausted: disposal.RetriesExhausted}, nil
	}
	a.retries++
	return pagevisitallowance.Allowance{Granted: true, PauseFor: time.Second}, nil
}

type fakeAcceptedOrders struct {
	profile yacycrawlcontract.CrawlProfile
	seeds   []canonicalurl.CanonicalURL
	err     error
}

func (o *fakeAcceptedOrders) OrderOf(
	_ context.Context, orderID string,
) (acceptedorder.AcceptedOrder, error) {
	if o.err != nil {
		return acceptedorder.AcceptedOrder{}, o.err
	}
	return acceptedorder.AcceptedOrderFrom(yacycrawlcontract.CrawlOrder{
		OrderID: orderID, Profile: o.profile, SeedURLs: o.seeds,
	})
}

type fakePendingVisits struct {
	mu        sync.Mutex
	published []pendingpagevisit.PendingPageVisit
	err       error
}

func (v *fakePendingVisits) Publish(
	_ context.Context,
	visit pendingpagevisit.PendingPageVisit,
) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.err != nil {
		return v.err
	}
	v.published = append(v.published, visit)
	return nil
}

func (v *fakePendingVisits) visits() []pendingpagevisit.PendingPageVisit {
	v.mu.Lock()
	defer v.mu.Unlock()
	return append([]pendingpagevisit.PendingPageVisit(nil), v.published...)
}

type fakePageVisitor struct {
	mu       sync.Mutex
	outcomes []pagevisit.PageVisitOutcome
	err      error
	visited  []string
}

func (f *fakePageVisitor) VisitPage(
	_ context.Context, url canonicalurl.CanonicalURL,
) (pagevisit.PageVisitOutcome, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.visited = append(f.visited, url.String())
	if f.err != nil {
		return pagevisit.PageVisitOutcome{}, f.err
	}
	if len(f.outcomes) == 0 {
		return pagevisit.PageVisitOutcome{Conclusion: pagevisit.PageVisitTerminal}, nil
	}
	outcome := f.outcomes[0]
	f.outcomes = f.outcomes[1:]
	return outcome, nil
}

func (f *fakePageVisitor) pageVisitCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.visited)
}

type recordingObserver struct {
	mu               sync.Mutex
	disposed         map[disposal.Reason]int
	deferrals        int
	retries          int
	returned         int
	claimedElsewhere int
}

func newObserver() *recordingObserver {
	return &recordingObserver{
		disposed: map[disposal.Reason]int{},
	}
}

func (o *recordingObserver) PendingPageVisitDisposedPage(
	_ context.Context,
	_ pendingpagevisit.PendingPageVisit,
	reason disposal.Reason,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.disposed[reason]++
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

func (o *recordingObserver) PendingPageVisitDroppedBecauseClaimedElsewhere(
	context.Context,
	pendingpagevisit.PendingPageVisit,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.claimedElsewhere++
}

func (*recordingObserver) PendingPageVisitCompleted(
	context.Context,
	pendingpagevisit.PendingPageVisit,
) {
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
		OrderID: orderID,
		URL:     canonicalurltest.CanonicalURLOf(t, visitURL),
		Depth:   0,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &pullintaketest.Message{Body: data, Sequence: sequence}
}

type crawlWorker struct {
	claims      *fakeClaims
	ledger      *fakeLedger
	orders      *fakeAcceptedOrders
	frontier    *fakePendingVisits
	pageVisitor *fakePageVisitor
	observer    *recordingObserver
}

func newWorker() *crawlWorker {
	return &crawlWorker{
		claims:      newClaims(),
		ledger:      newLedger(),
		orders:      &fakeAcceptedOrders{profile: wideProfile()},
		frontier:    &fakePendingVisits{},
		pageVisitor: &fakePageVisitor{},
		observer:    newObserver(),
	}
}

func (w *crawlWorker) consume(t *testing.T, messages ...jetstream.Msg) error {
	t.Helper()
	return pagevisitintake.NewPageVisitConsumer(
		pullintaketest.MessageSourceOf(messages...),
		w.claims,
		w.ledger,
		w.orders,
		w.frontier,
		w.pageVisitor,
		w.observer,
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

func TestDiscoveredURLsTheProfileAdmitsGoBackOnTheFrontier(t *testing.T) {
	worker := newWorker()
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{{
		Conclusion: pagevisit.PageVisitTerminal,
		DiscoveredURLs: []canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, "http://host/next"),
		},
	}}

	if err := worker.consume(t, visitMessage(t, 1)); err != nil {
		t.Fatalf("run: %v", err)
	}

	published := worker.frontier.visits()
	if len(published) != 1 {
		t.Fatalf("published %d urls, want 1", len(published))
	}
	if published[0].Depth != 1 || published[0].OrderID != orderID {
		t.Fatalf("published %+v, want the order at depth one", published[0])
	}
}

func TestDiscoveredURLsBeyondTheProfileStayOff(t *testing.T) {
	worker := newWorker()
	worker.orders.profile.MaxDepth = 0
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{{
		Conclusion: pagevisit.PageVisitTerminal,
		DiscoveredURLs: []canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, "http://host/next"),
		},
	}}

	if err := worker.consume(t, visitMessage(t, 1)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(worker.frontier.visits()) != 0 {
		t.Fatal("a url beyond the profile depth should not be published")
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
	worker.ledger.deferrals = 2
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{
		{Conclusion: pagevisit.PageVisitDeferred},
	}
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.observer.disposed[disposal.DeferralsExhausted] != 1 {
		t.Fatalf("observer disposed %v, want deferrals exhausted", worker.observer.disposed)
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

func TestAURLThatExhaustedItsRetriesIsDropped(t *testing.T) {
	worker := newWorker()
	worker.ledger.retries = 2
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{
		{Conclusion: pagevisit.PageVisitRetryable},
	}

	if err := worker.consume(t, visitMessage(t, 1)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.observer.disposed[disposal.RetriesExhausted] != 1 {
		t.Fatalf("observer disposed %v, want retries exhausted", worker.observer.disposed)
	}
}

func TestAURLWhoseHostExhaustedItsPagesIsDroppedBeforeTheFetch(t *testing.T) {
	worker := newWorker()
	worker.ledger.hostSpent = false
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.pageVisitor.pageVisitCount() != 0 {
		t.Fatal("a host that spent its pages should not be fetched again")
	}
	if worker.observer.disposed[disposal.HostPagesExhausted] != 1 {
		t.Fatalf("observer disposed %v, want host pages exhausted", worker.observer.disposed)
	}
}

func TestTheProfilesHostPageLimitReachesTheLedger(t *testing.T) {
	worker := newWorker()
	worker.orders.profile.MaxPagesPerHost = 7

	if err := worker.consume(t, visitMessage(t, 1)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(worker.ledger.hostPageLimits) != 1 || worker.ledger.hostPageLimits[0] != 7 {
		t.Fatalf(
			"ledger spent against %v, want the profile limit of 7",
			worker.ledger.hostPageLimits,
		)
	}
}

func TestARedeliveredMessageAsksTheLedgerForItsHostPageAgain(t *testing.T) {
	worker := newWorker()
	message := visitMessage(t, 1)

	if err := worker.consume(t, message, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(worker.ledger.hostPageLimits) != 2 {
		t.Fatalf(
			"the ledger was asked for %d host pages, want one for each delivery",
			len(worker.ledger.hostPageLimits),
		)
	}
}

func TestAVisitThatFailsReturnsForRedelivery(t *testing.T) {
	worker := newWorker()
	worker.pageVisitor.err = errors.New("fetch exploded")
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.HeldBack {
		t.Fatalf("message settled %v, want one delayed return", got)
	}
}

func TestAVisitWhoseClaimTheLedgerCannotAnswerReturnsForRedelivery(t *testing.T) {
	worker := newWorker()
	worker.claims.claimErr = errors.New("bucket down")
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

func TestTheDisposalTheVisitReportsIsObserved(t *testing.T) {
	worker := newWorker()
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{{
		Conclusion: pagevisit.PageVisitTerminal, Disposal: disposal.FetchRejected,
	}}

	if err := worker.consume(t, visitMessage(t, 1)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.observer.disposed[disposal.FetchRejected] != 1 {
		t.Fatalf("observer disposed %v, want fetch rejected", worker.observer.disposed)
	}
}

func TestDiscoveredURLsThatDoNotPublishReturnTheMessage(t *testing.T) {
	worker := newWorker()
	worker.frontier.err = errors.New("stream down")
	worker.pageVisitor.outcomes = []pagevisit.PageVisitOutcome{{
		Conclusion: pagevisit.PageVisitTerminal,
		DiscoveredURLs: []canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, "http://host/next"),
		},
	}}
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.HeldBack {
		t.Fatalf("message settled %v, want one delayed return", got)
	}
}
