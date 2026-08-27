package visitintake_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitintake"
)

const (
	orderID  = "o1"
	visitURL = "http://host/"
)

type fakeMsg struct {
	data      []byte
	delivered uint64
	mu        sync.Mutex
	settlings []string
	nakDelay  time.Duration
}

func (m *fakeMsg) Subject() string                 { return "yacy.crawl.frontier" }
func (m *fakeMsg) Reply() string                   { return "" }
func (m *fakeMsg) Data() []byte                    { return m.data }
func (m *fakeMsg) Headers() nats.Header            { return nil }
func (m *fakeMsg) Ack() error                      { return m.settle("ack") }
func (m *fakeMsg) DoubleAck(context.Context) error { return m.settle("ack") }
func (m *fakeMsg) Nak() error                      { return m.settle("nak") }
func (m *fakeMsg) InProgress() error               { return nil }
func (m *fakeMsg) Term() error                     { return m.settle("term") }
func (m *fakeMsg) TermWithReason(string) error     { return m.settle("term") }

func (m *fakeMsg) NakWithDelay(delay time.Duration) error {
	m.mu.Lock()
	m.nakDelay = delay
	m.mu.Unlock()
	return m.settle("nak-with-delay")
}

func (m *fakeMsg) Metadata() (*jetstream.MsgMetadata, error) {
	return &jetstream.MsgMetadata{NumDelivered: m.delivered}, nil
}

func (m *fakeMsg) settle(action string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settlings = append(m.settlings, action)
	return nil
}

func (m *fakeMsg) settled() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.settlings...)
}

func (m *fakeMsg) heldBackFor() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.nakDelay
}

type fakeIterator struct {
	mu       sync.Mutex
	messages []jetstream.Msg
}

func (it *fakeIterator) Next(...jetstream.NextOpt) (jetstream.Msg, error) {
	it.mu.Lock()
	defer it.mu.Unlock()
	if len(it.messages) == 0 {
		return nil, jetstream.ErrMsgIteratorClosed
	}
	msg := it.messages[0]
	it.messages = it.messages[1:]
	return msg, nil
}

func (it *fakeIterator) Stop()  {}
func (it *fakeIterator) Drain() {}

type fakeSource struct{ iterator *fakeIterator }

func (s fakeSource) Messages(...jetstream.PullMessagesOpt) (jetstream.MessagesContext, error) {
	return s.iterator, nil
}

type fakeClaims struct {
	claimed      map[string]bool
	claimErr     error
	hostSpent    bool
	deferrals    int
	maxDeferrals int
	attempts     int
	maxAttempts  int
}

func newClaims() *fakeClaims {
	return &fakeClaims{
		claimed:      map[string]bool{},
		hostSpent:    true,
		maxDeferrals: 2,
		maxAttempts:  2,
	}
}

func (c *fakeClaims) Claim(
	_ context.Context, _ string, url canonicalurl.CanonicalURL,
) (bool, error) {
	if c.claimErr != nil {
		return false, c.claimErr
	}
	if c.claimed[url.String()] {
		return false, nil
	}
	c.claimed[url.String()] = true
	return true, nil
}

func (c *fakeClaims) SpendHostPage(_ context.Context, _ string, _ string, _ int) (bool, error) {
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
	if c.attempts >= c.maxAttempts {
		return 0, false, nil
	}
	c.attempts++
	return c.attempts, true, nil
}

type fakeAcceptedOrders struct {
	profile yacycrawlcontract.CrawlProfile
	seeds   []canonicalurl.CanonicalURL
	err     error
}

func (o *fakeAcceptedOrders) OrderOf(
	_ context.Context, orderID string,
) (yacycrawlcontract.CrawlOrder, error) {
	if o.err != nil {
		return yacycrawlcontract.CrawlOrder{}, o.err
	}
	return yacycrawlcontract.CrawlOrder{
		OrderID: orderID, Profile: o.profile, SeedURLs: o.seeds,
	}, nil
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
	if len(f.outcomes) > 1 {
		f.outcomes = f.outcomes[1:]
	}
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

func visitMessage(t *testing.T, delivered uint64) *fakeMsg {
	t.Helper()
	data, err := pendingvisit.MarshalPendingVisit(pendingvisit.PendingVisit{
		OrderID: orderID,
		URL:     canonicalurltest.CanonicalURLOf(t, visitURL),
		Depth:   0,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &fakeMsg{data: data, delivered: delivered}
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
		Source:           fakeSource{iterator: &fakeIterator{messages: messages}},
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
	if got := message.settled(); len(got) != 1 || got[0] != "ack" {
		t.Fatalf("message settled %v, want one ack", got)
	}
}

func TestAURLAnotherWorkerClaimedIsDroppedOnItsFirstDelivery(t *testing.T) {
	worker := newWorker()
	worker.claims.claimed[visitURL] = true
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.visitor.visitCount() != 0 {
		t.Fatal("a url another worker claimed should not be visited")
	}
	if got := message.settled(); len(got) != 1 || got[0] != "ack" {
		t.Fatalf("message settled %v, want one ack", got)
	}
}

func TestARedeliveredMessageVisitsItsOwnClaim(t *testing.T) {
	worker := newWorker()
	worker.claims.claimed[visitURL] = true
	message := visitMessage(t, 2)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.visitor.visitCount() != 1 {
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

	if got := message.settled(); len(got) != 1 || got[0] != "nak-with-delay" {
		t.Fatalf("message settled %v, want one delayed return", got)
	}
	if message.heldBackFor() != 7*time.Second {
		t.Fatalf("held back for %v, want the delay the site asked", message.heldBackFor())
	}
	if worker.observer.refusals[refusal.Defer] != 1 {
		t.Fatalf("observer refusals %v, want one defer", worker.observer.refusals)
	}
}

func TestAURLThatSpentItsDeferralsIsDisposed(t *testing.T) {
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
	if got := message.settled(); len(got) != 1 || got[0] != "ack" {
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

	if message.heldBackFor() != time.Second {
		t.Fatalf("held back for %v, want the first backoff", message.heldBackFor())
	}
}

func TestAURLThatSpentItsAttemptsIsDisposed(t *testing.T) {
	worker := newWorker()
	worker.claims.attempts = 2
	worker.visitor.outcomes = []pagevisit.VisitOutcome{{Conclusion: pagevisit.VisitRetryable}}

	if err := worker.consume(t, visitMessage(t, 1)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.observer.disposed[disposal.FetchAbandoned] != 1 {
		t.Fatalf("observer disposed %v, want fetch abandoned", worker.observer.disposed)
	}
}

func TestAURLWhoseHostSpentItsPagesIsDisposedBeforeTheFetch(t *testing.T) {
	worker := newWorker()
	worker.claims.hostSpent = false
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.visitor.visitCount() != 0 {
		t.Fatal("a host that spent its pages should not be fetched again")
	}
	if worker.observer.disposed[disposal.HostPagesSpent] != 1 {
		t.Fatalf("observer disposed %v, want host pages spent", worker.observer.disposed)
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

	if got := message.settled(); len(got) != 1 || got[0] != "nak" {
		t.Fatalf("message settled %v, want one nak", got)
	}
}

func TestAVisitOfAnUnreadableOrderReturnsForRedelivery(t *testing.T) {
	worker := newWorker()
	worker.orders.err = errors.New("bucket down")
	message := visitMessage(t, 1)

	if err := worker.consume(t, message); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := message.settled(); len(got) != 1 || got[0] != "nak" {
		t.Fatalf("message settled %v, want one nak", got)
	}
}

func TestAnUndecodablePendingVisitHaltsIntake(t *testing.T) {
	if err := newWorker().consume(t, &fakeMsg{data: []byte("{"), delivered: 1}); err == nil {
		t.Fatal("an undecodable pending visit should halt intake")
	}
}

func TestTheDisposalTheVisitReportsIsObserved(t *testing.T) {
	worker := newWorker()
	worker.visitor.outcomes = []pagevisit.VisitOutcome{{
		Conclusion: pagevisit.VisitCompleted, Disposal: disposal.NotAPage,
	}}

	if err := worker.consume(t, visitMessage(t, 1)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if worker.observer.disposed[disposal.NotAPage] != 1 {
		t.Fatalf("observer disposed %v, want not-a-page", worker.observer.disposed)
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

	if got := message.settled(); len(got) != 1 || got[0] != "nak" {
		t.Fatalf("message settled %v, want one nak", got)
	}
}
