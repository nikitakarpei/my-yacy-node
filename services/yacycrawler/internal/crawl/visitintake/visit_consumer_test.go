package visitintake_test

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
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitclaim"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitintake"
)

const (
	orderID  = "o1"
	visitURL = "http://host/"
)

type fakeClaims struct {
	holders        map[string]string
	claimErr       error
	hostSpent      bool
	hostPageLimits []int
	deferrals      int
	maxDeferrals   int
	retries        int
	maxRetries     int
}

func newClaims() *fakeClaims {
	return &fakeClaims{
		holders:      map[string]string{},
		hostSpent:    true,
		maxDeferrals: 2,
		maxRetries:   2,
	}
}

func (c *fakeClaims) Claim(
	_ context.Context, _ string, url canonicalurl.CanonicalURL, holder string,
) (visitclaim.Claim, error) {
	if c.claimErr != nil {
		return visitclaim.Unanswered, c.claimErr
	}
	standing, held := c.holders[url.String()]
	if !held {
		c.holders[url.String()] = holder
		return visitclaim.Taken, nil
	}
	if standing != holder {
		return visitclaim.HeldElsewhere, nil
	}
	return visitclaim.Resumed, nil
}

func (c *fakeClaims) SpendHostPage(
	_ context.Context, _ string, _ string, maxPages int,
) (bool, error) {
	c.hostPageLimits = append(c.hostPageLimits, maxPages)
	return c.hostSpent, nil
}

func (c *fakeClaims) Defer(_ context.Context, _ string, _ canonicalurl.CanonicalURL) (bool, error) {
	if c.deferrals >= c.maxDeferrals {
		return false, nil
	}
	c.deferrals++
	return true, nil
}

func (c *fakeClaims) Retry(
	_ context.Context, _ string, _ canonicalurl.CanonicalURL,
) (int, bool, error) {
	if c.retries >= c.maxRetries {
		return 0, false, nil
	}
	c.retries++
	return c.retries, true, nil
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
	published []pendingvisit.PendingVisit
	err       error
}

func (v *fakePendingVisits) Publish(_ context.Context, visit pendingvisit.PendingVisit) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.err != nil {
		return v.err
	}
	v.published = append(v.published, visit)
	return nil
}

func (v *fakePendingVisits) visits() []pendingvisit.PendingVisit {
	v.mu.Lock()
	defer v.mu.Unlock()
	return append([]pendingvisit.PendingVisit(nil), v.published...)
}

type fakeVisitor struct {
	mu       sync.Mutex
	outcomes []pagevisit.VisitOutcome
	err      error
	visited  []string
	refusals []pagevisit.IgnoredRefusals
}

func (f *fakeVisitor) visitorFor(ignoredRefusals pagevisit.IgnoredRefusals) pagevisit.Visitor {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refusals = append(f.refusals, ignoredRefusals)
	return f
}

func (f *fakeVisitor) Visit(
	_ context.Context, url canonicalurl.CanonicalURL,
) (pagevisit.VisitOutcome, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.visited = append(f.visited, url.String())
	if f.err != nil {
		return pagevisit.VisitOutcome{}, f.err
	}
	if len(f.outcomes) == 0 {
		return pagevisit.VisitOutcome{Conclusion: pagevisit.VisitCompleted}, nil
	}
	outcome := f.outcomes[0]
	f.outcomes = f.outcomes[1:]
	return outcome, nil
}

func (f *fakeVisitor) visitCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.visited)
}

type recordingObserver struct {
	mu       sync.Mutex
	disposed map[disposal.Reason]int
	refusals map[refusal.Demand]int
}

func newObserver() *recordingObserver {
	return &recordingObserver{
		disposed: map[disposal.Reason]int{},
		refusals: map[refusal.Demand]int{},
	}
}

func (o *recordingObserver) PageDisposed(reason disposal.Reason) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.disposed[reason]++
}

func (o *recordingObserver) RefusalHonored(demand refusal.Demand) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.refusals[demand]++
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
	data, err := pendingvisit.MarshalPendingVisit(pendingvisit.PendingVisit{
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
	claims   *fakeClaims
	orders   *fakeAcceptedOrders
	visits   *fakePendingVisits
	visitor  *fakeVisitor
	observer *recordingObserver
}

func newWorker() *crawlWorker {
	return &crawlWorker{
		claims:   newClaims(),
		orders:   &fakeAcceptedOrders{profile: wideProfile()},
		visits:   &fakePendingVisits{},
		visitor:  &fakeVisitor{},
		observer: newObserver(),
	}
}

func (w *crawlWorker) consume(t *testing.T, messages ...jetstream.Msg) error {
	t.Helper()
	return visitintake.NewVisitConsumer(visitintake.Config{
		Source:           pullintaketest.MessageSourceOf(messages...),
		Claims:           w.claims,
		Orders:           w.orders,
		Visits:           w.visits,
		VisitorFor:       w.visitor.visitorFor,
		Observer:         w.observer,
		RetryDelay:       retrydelay.Bounds{Floor: time.Second, Ceiling: time.Minute},
		FetchConcurrency: 1,
	}).Run(context.Background())
}

func TestAClaimedURLIsVisitedThenAcknowledged(t *testing.T) {
	worker := newWorker()
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.visitor.visitCount() != 1 {
		t.Fatalf("visited %d urls, want 1", worker.visitor.visitCount())
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

	if worker.visitor.visitCount() != 1 {
		t.Fatalf("visited %d times, want the url fetched once", worker.visitor.visitCount())
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

	if worker.visitor.visitCount() != 2 {
		t.Fatal("a redelivery should visit the claim it left behind")
	}
}

func TestDiscoveredURLsTheProfileAdmitsGoBackOnTheFrontier(t *testing.T) {
	worker := newWorker()
	worker.visitor.outcomes = []pagevisit.VisitOutcome{{
		Conclusion: pagevisit.VisitCompleted,
		DiscoveredURLs: []canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, "http://host/next"),
		},
	}}

	if err := worker.consume(t, visitMessage(t, 1)); err != nil {
		t.Fatalf("run: %v", err)
	}

	published := worker.visits.visits()
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
	worker.visitor.outcomes = []pagevisit.VisitOutcome{{
		Conclusion: pagevisit.VisitCompleted,
		DiscoveredURLs: []canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, "http://host/next"),
		},
	}}

	if err := worker.consume(t, visitMessage(t, 1)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(worker.visits.visits()) != 0 {
		t.Fatal("a url beyond the profile depth should not be published")
	}
}

func TestADeferredVisitReturnsAfterTheDelayTheSiteAsked(t *testing.T) {
	worker := newWorker()
	worker.visitor.outcomes = []pagevisit.VisitOutcome{{
		Conclusion: pagevisit.VisitDeferred, DeferFor: 7 * time.Second,
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
	if worker.observer.refusals[refusal.Defer] != 1 {
		t.Fatalf("observer refusals %v, want one defer", worker.observer.refusals)
	}
}

func TestAURLThatExhaustedItsDeferralsIsDropped(t *testing.T) {
	worker := newWorker()
	worker.claims.deferrals = 2
	worker.visitor.outcomes = []pagevisit.VisitOutcome{{Conclusion: pagevisit.VisitDeferred}}
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
	worker.visitor.outcomes = []pagevisit.VisitOutcome{{Conclusion: pagevisit.VisitRetryable}}
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
	worker.claims.retries = 2
	worker.visitor.outcomes = []pagevisit.VisitOutcome{{Conclusion: pagevisit.VisitRetryable}}

	if err := worker.consume(t, visitMessage(t, 1)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.observer.disposed[disposal.RetriesExhausted] != 1 {
		t.Fatalf("observer disposed %v, want retries exhausted", worker.observer.disposed)
	}
}

func TestAURLWhoseHostExhaustedItsPagesIsDroppedBeforeTheFetch(t *testing.T) {
	worker := newWorker()
	worker.claims.hostSpent = false
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.visitor.visitCount() != 0 {
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

	if len(worker.claims.hostPageLimits) != 1 || worker.claims.hostPageLimits[0] != 7 {
		t.Fatalf(
			"ledger spent against %v, want the profile limit of 7",
			worker.claims.hostPageLimits,
		)
	}
}

func TestARedeliveredMessageSpendsNoFurtherHostPage(t *testing.T) {
	worker := newWorker()
	message := visitMessage(t, 1)

	if err := worker.consume(t, message, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(worker.claims.hostPageLimits) != 1 {
		t.Fatalf(
			"the ledger spent %d host pages, want the one the first delivery took",
			len(worker.claims.hostPageLimits),
		)
	}
}

func TestTheVisitorHonorsTheOrdersIndexingRefusal(t *testing.T) {
	worker := newWorker()
	worker.orders.profile.IgnoresIndexingRefusal = true

	if err := worker.consume(t, visitMessage(t, 1)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(worker.visitor.refusals) != 1 ||
		!worker.visitor.refusals[0].IndexingRefusal {
		t.Fatalf("visitor built for %v, want ignored", worker.visitor.refusals)
	}
}

func TestAVisitThatFailsReturnsForRedelivery(t *testing.T) {
	worker := newWorker()
	worker.visitor.err = errors.New("fetch exploded")
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

	if worker.visitor.visitCount() != 0 {
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
	if worker.visitor.visitCount() != 0 {
		t.Fatal("a url under an unreadable profile should not be visited")
	}
}

func TestAnUndecodablePendingVisitHaltsIntake(t *testing.T) {
	if err := newWorker().consume(t, &pullintaketest.Message{Body: []byte("{"), Sequence: 1}); err == nil {
		t.Fatal("an undecodable pending visit should halt intake")
	}
}

func TestTheDisposalTheVisitReportsIsObserved(t *testing.T) {
	worker := newWorker()
	worker.visitor.outcomes = []pagevisit.VisitOutcome{{
		Conclusion: pagevisit.VisitCompleted, Disposal: disposal.FetchRejected,
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
	worker.visits.err = errors.New("stream down")
	worker.visitor.outcomes = []pagevisit.VisitOutcome{{
		Conclusion: pagevisit.VisitCompleted,
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
