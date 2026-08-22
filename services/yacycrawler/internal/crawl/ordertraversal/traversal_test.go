package ordertraversal_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/frontier"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/ordertraversal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
)

type fakeVisitor struct {
	mu      sync.Mutex
	queued  map[string][]pagevisit.VisitOutcome
	err     error
	visited map[string]int
}

func (f *fakeVisitor) Visit(
	_ context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) (pagevisit.VisitOutcome, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.visited == nil {
		f.visited = map[string]int{}
	}
	f.visited[canonicalURL.String()]++
	if f.err != nil {
		return pagevisit.VisitOutcome{}, f.err
	}
	queue := f.queued[canonicalURL.String()]
	if len(queue) == 0 {
		return fetchedPage(), nil
	}
	outcome := queue[0]
	if len(queue) > 1 {
		f.queued[canonicalURL.String()] = queue[1:]
	}
	return outcome, nil
}

func (f *fakeVisitor) visitCount(url string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.visited[url]
}

func fetchedPage() pagevisit.VisitOutcome {
	return pagevisit.VisitOutcome{Conclusion: pagevisit.VisitCompleted, Fetched: true}
}

type recordingObserver struct {
	mu       sync.Mutex
	disposed map[disposal.Reason]int
	refusals map[refusal.Demand]int
	budget   int
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

func (o *recordingObserver) RefusalHonored(kind refusal.Demand) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.refusals[kind]++
}

func (o *recordingObserver) BudgetExhausted() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.budget++
}

type manualClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *manualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *manualClock) Sleep(ctx context.Context, duration time.Duration) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("manual clock: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(duration)
	return nil
}

func defaultConfig() ordertraversal.Config {
	return ordertraversal.Config{
		RunPageBudget:    100,
		VisitConcurrency: 1,
		MaxAdmittedURLs:  100,
		Frontier: frontier.Config{
			MaxDeferralsPerURL: 2,
			MaxAttemptsPerURL:  2,
			RetryDelay: retrydelay.Bounds{
				Floor:   time.Millisecond,
				Ceiling: time.Millisecond,
			},
		},
	}
}

func wideProfile() yacycrawlcontract.CrawlProfile {
	return yacycrawlcontract.CrawlProfile{
		Scope:           yacycrawlcontract.ScopeWide,
		URLMustMatch:    yacycrawlcontract.MatchAll,
		MaxDepth:        5,
		MaxPagesPerHost: yacycrawlcontract.UnlimitedPagesPerHost,
	}
}

func order(seeds []canonicalurl.CanonicalURL) yacycrawlcontract.CrawlOrder {
	return yacycrawlcontract.CrawlOrder{
		OrderID: "o1", Profile: wideProfile(), SeedURLs: seeds,
	}
}

type fixedVisitorSource struct {
	visitor          pagevisit.Visitor
	indexingRefusals []pagevisit.IndexingRefusal
}

func (s *fixedVisitorSource) VisitorFor(
	indexingRefusal pagevisit.IndexingRefusal,
) pagevisit.Visitor {
	s.indexingRefusals = append(s.indexingRefusals, indexingRefusal)
	return s.visitor
}

func newTraverser(
	config ordertraversal.Config,
	visitor pagevisit.Visitor,
	observer *recordingObserver,
) *ordertraversal.Traverser {
	return ordertraversal.New(
		config,
		&fixedVisitorSource{visitor: visitor},
		observer,
		&manualClock{},
	)
}

func traverse(
	t *testing.T,
	traverser *ordertraversal.Traverser,
	seeds []canonicalurl.CanonicalURL,
) {
	t.Helper()
	if err := traverser.Traverse(context.Background(), order(seeds)); err != nil {
		t.Fatalf("traverse: %v", err)
	}
}

func TestTraverseTakesItsVisitorFromTheOrdersIndexingRefusal(t *testing.T) {
	observer := newObserver()
	visitors := &fixedVisitorSource{
		visitor: &fakeVisitor{queued: map[string][]pagevisit.VisitOutcome{}},
	}
	traverser := ordertraversal.New(
		defaultConfig(),
		visitors,
		observer,
		&manualClock{},
	)

	ignoring := order(
		[]canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://host/")},
	)
	ignoring.Profile.IgnoresIndexingRefusal = true
	if err := traverser.Traverse(context.Background(), ignoring); err != nil {
		t.Fatalf("traverse ignoring order: %v", err)
	}
	if err := traverser.Traverse(
		context.Background(),
		order([]canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://host/")}),
	); err != nil {
		t.Fatalf("traverse honoring order: %v", err)
	}

	want := []pagevisit.IndexingRefusal{pagevisit.Ignored, pagevisit.Honored}
	if !slices.Equal(visitors.indexingRefusals, want) {
		t.Fatalf("visitors built for %v, want %v", visitors.indexingRefusals, want)
	}
}

func TestTraverseDiscoversAndCrawlsLinks(t *testing.T) {
	visitor := &fakeVisitor{queued: map[string][]pagevisit.VisitOutcome{
		"http://host/": {
			{
				Conclusion: pagevisit.VisitCompleted,
				Fetched:    true,
				DiscoveredURLs: []canonicalurl.CanonicalURL{
					canonicalurltest.CanonicalURLOf(t, "http://host/next"),
				},
			},
		},
	}}
	traverser := newTraverser(defaultConfig(), visitor, newObserver())

	traverse(
		t,
		traverser,
		[]canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://host/")},
	)

	if visitor.visitCount("http://host/next") != 1 {
		t.Fatalf("want discovered link visited, got counts %v", visitor.visited)
	}
}

func TestTraverseDisposesTheVisitsReportedReason(t *testing.T) {
	visitor := &fakeVisitor{queued: map[string][]pagevisit.VisitOutcome{
		"http://host/": {{
			Conclusion: pagevisit.VisitCompleted,
			Fetched:    true,
			Disposal:   disposal.NotAPage,
		}},
	}}
	observer := newObserver()
	traverser := newTraverser(defaultConfig(), visitor, observer)

	traverse(
		t,
		traverser,
		[]canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://host/")},
	)

	if observer.disposed[disposal.NotAPage] != 1 {
		t.Fatalf("want the visit's reason observed, got %v", observer.disposed)
	}
}

func TestTraverseRecordsAPublishedPageAsNotDisposed(t *testing.T) {
	visitor := &fakeVisitor{queued: map[string][]pagevisit.VisitOutcome{}}
	observer := newObserver()
	traverser := newTraverser(defaultConfig(), visitor, observer)

	traverse(
		t,
		traverser,
		[]canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://host/")},
	)

	if len(observer.disposed) != 0 {
		t.Fatalf("a published page must not be observed disposed, got %v", observer.disposed)
	}
}

func TestTraverseBudgetTruncatesAndRecordsTheRemainder(t *testing.T) {
	cfg := defaultConfig()
	cfg.RunPageBudget = 1
	visitor := &fakeVisitor{queued: map[string][]pagevisit.VisitOutcome{
		"http://host/": {
			{
				Conclusion: pagevisit.VisitCompleted,
				Fetched:    true,
				DiscoveredURLs: []canonicalurl.CanonicalURL{
					canonicalurltest.CanonicalURLOf(t, "http://host/a"),
					canonicalurltest.CanonicalURLOf(t, "http://host/b"),
				},
			},
		},
	}}
	observer := newObserver()
	traverser := newTraverser(cfg, visitor, observer)

	traverse(
		t,
		traverser,
		[]canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://host/")},
	)

	if observer.budget != 1 || observer.disposed[disposal.BudgetTruncated] != 2 {
		t.Fatalf("budget not exhausted: budget=%d disposed=%v", observer.budget, observer.disposed)
	}
}

func TestTraverseBudgetCountsOnlyFetches(t *testing.T) {
	cfg := defaultConfig()
	cfg.RunPageBudget = 1
	visitor := &fakeVisitor{queued: map[string][]pagevisit.VisitOutcome{
		"http://host/a": {{Conclusion: pagevisit.VisitCompleted, Disposal: disposal.NotDue}},
	}}
	observer := newObserver()
	traverser := newTraverser(cfg, visitor, observer)

	traverse(
		t,
		traverser,
		[]canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, "http://host/a"),
			canonicalurltest.CanonicalURLOf(t, "http://host/b"),
		},
	)

	if visitor.visitCount("http://host/b") != 1 {
		t.Fatal("a skipped fetch leaves the budget for the next seed")
	}
	if observer.disposed[disposal.BudgetTruncated] != 0 {
		t.Fatalf("nothing should be truncated, got %v", observer.disposed)
	}
}

func TestTraverseBudgetCountsAFetchedPage(t *testing.T) {
	cfg := defaultConfig()
	cfg.RunPageBudget = 1
	visitor := &fakeVisitor{queued: map[string][]pagevisit.VisitOutcome{
		"http://host/a": {fetchedPage()},
	}}
	observer := newObserver()
	traverser := newTraverser(cfg, visitor, observer)

	traverse(
		t,
		traverser,
		[]canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, "http://host/a"),
			canonicalurltest.CanonicalURLOf(t, "http://host/b"),
		},
	)

	if observer.disposed[disposal.BudgetTruncated] != 1 {
		t.Fatalf("a fetched page should consume the budget, got %v", observer.disposed)
	}
	if visitor.visitCount("http://host/b") != 0 {
		t.Fatal("budget was exhausted, the second seed should not be visited")
	}
}

func TestTraverseDefersThenGivesUp(t *testing.T) {
	deferred := pagevisit.VisitOutcome{Conclusion: pagevisit.VisitDeferred, DeferFor: time.Second}
	visitor := &fakeVisitor{queued: map[string][]pagevisit.VisitOutcome{
		"http://host/": {deferred, deferred, deferred, deferred},
	}}
	observer := newObserver()
	traverser := newTraverser(defaultConfig(), visitor, observer)

	traverse(
		t,
		traverser,
		[]canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://host/")},
	)

	if observer.refusals[refusal.Defer] == 0 {
		t.Fatal("expected defer refusals")
	}
	if observer.disposed[disposal.DeferralsExhausted] != 1 {
		t.Fatalf("expected deferrals-exhausted after defer limit, got %v", observer.disposed)
	}
}

func TestTraverseRetriesTransientFetchThenSucceeds(t *testing.T) {
	transient := pagevisit.VisitOutcome{Conclusion: pagevisit.VisitRetryable}
	visitor := &fakeVisitor{queued: map[string][]pagevisit.VisitOutcome{
		"http://host/": {transient, transient, fetchedPage()},
	}}
	traverser := newTraverser(defaultConfig(), visitor, newObserver())

	traverse(
		t,
		traverser,
		[]canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://host/")},
	)

	if visitor.visitCount("http://host/") != 3 {
		t.Fatalf(
			"want 3 visits (2 transient + success), got %d",
			visitor.visitCount("http://host/"),
		)
	}
}

func TestTraverseAbandonsTransientFetchAfterLimit(t *testing.T) {
	transient := pagevisit.VisitOutcome{Conclusion: pagevisit.VisitRetryable}
	visitor := &fakeVisitor{queued: map[string][]pagevisit.VisitOutcome{
		"http://host/": {transient, transient, transient},
	}}
	observer := newObserver()
	traverser := newTraverser(defaultConfig(), visitor, observer)

	traverse(
		t,
		traverser,
		[]canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://host/")},
	)

	if observer.disposed[disposal.FetchAbandoned] != 1 {
		t.Fatalf("expected fetch-abandoned after retry limit, got %v", observer.disposed)
	}
}

func TestTraverseVisitorErrorFails(t *testing.T) {
	visitor := &fakeVisitor{err: errors.New("boom")}
	traverser := newTraverser(defaultConfig(), visitor, newObserver())

	if err := traverser.Traverse(
		context.Background(),
		order([]canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://host/")}),
	); err == nil {
		t.Fatal("visitor error should fail the traversal")
	}
}
