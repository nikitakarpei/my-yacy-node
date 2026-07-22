package ordertraversal_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/ordertraversal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisit"
)

type fakeVisitor struct {
	mu      sync.Mutex
	queued  map[string][]pagevisit.VisitOutcome
	err     error
	visited map[string]int
}

func (f *fakeVisitor) Visit(_ context.Context, url string) (pagevisit.VisitOutcome, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.visited == nil {
		f.visited = map[string]int{}
	}
	f.visited[url]++
	if f.err != nil {
		return pagevisit.VisitOutcome{}, f.err
	}
	queue := f.queued[url]
	if len(queue) == 0 {
		return pagevisit.VisitOutcome{Classification: pagevisit.Succeeded}, nil
	}
	outcome := queue[0]
	if len(queue) > 1 {
		f.queued[url] = queue[1:]
	}
	return outcome, nil
}

func (f *fakeVisitor) visitCount(url string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.visited[url]
}

type recordingObserver struct {
	mu       sync.Mutex
	disposed map[string]int
	refusals map[string]int
	budget   int
}

func newObserver() *recordingObserver {
	return &recordingObserver{disposed: map[string]int{}, refusals: map[string]int{}}
}

func (*recordingObserver) OrderReceived()              {}
func (*recordingObserver) OrderRedelivered()           {}
func (*recordingObserver) OrderCompleted()             {}
func (*recordingObserver) PageFetched()                {}
func (*recordingObserver) PagePublished(string)        {}
func (*recordingObserver) PublicationWaited()          {}
func (*recordingObserver) FetchObserved(time.Duration) {}

func (o *recordingObserver) PageDisposed(reason string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.disposed[reason]++
}

func (o *recordingObserver) RefusalHonored(kind string) {
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

func (c *manualClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("manual clock: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	return nil
}

func defaultConfig() ordertraversal.Config {
	return ordertraversal.Config{
		RunPageBudget:       100,
		FrontierCapacity:    100,
		FetchRetryLimit:     2,
		FetchRetryFloor:     time.Millisecond,
		FetchRetryCeiling:   time.Millisecond,
		PublishRetryFloor:   time.Millisecond,
		PublishRetryCeiling: time.Millisecond,
		MaxDeferralsPerURL:  2,
		FetchConcurrency:    1,
	}
}

func wideProfile() yacycrawlcontract.CrawlProfile {
	return yacycrawlcontract.NewCrawlProfile(yacycrawlcontract.CrawlProfile{
		Scope:           yacycrawlcontract.ScopeWide,
		URLMustMatch:    yacycrawlcontract.MatchAll,
		MaxDepth:        5,
		MaxPagesPerHost: yacycrawlcontract.UnlimitedPagesPerHost,
	})
}

func orderDelivery(seeds []string) crawlcapability.DeliveredOrder {
	return crawlcapability.DeliveredOrder{
		Order: yacycrawlcontract.CrawlOrder{
			OrderID: "o1", Profile: wideProfile(), SeedURLs: seeds,
		},
		Ack:             func(context.Context) error { return nil },
		Retry:           func(context.Context) error { return nil },
		ExtendOwnership: func(context.Context) error { return nil },
	}
}

func traverse(t *testing.T, traverser *ordertraversal.OrderTraverser, seeds []string) {
	t.Helper()
	if err := traverser.Traverse(context.Background(), orderDelivery(seeds)); err != nil {
		t.Fatalf("traverse: %v", err)
	}
}

func TestTraverseDiscoversAndCrawlsLinks(t *testing.T) {
	visitor := &fakeVisitor{queued: map[string][]pagevisit.VisitOutcome{
		"http://host/": {
			{Classification: pagevisit.Succeeded, DiscoveredURLs: []string{"http://host/next"}},
		},
	}}
	traverser := ordertraversal.NewOrderTraverser(
		defaultConfig(),
		visitor,
		newObserver(),
		&manualClock{},
	)

	traverse(t, traverser, []string{"http://host/"})

	if visitor.visitCount("http://host/next") != 1 {
		t.Fatalf("want discovered link visited, got counts %v", visitor.visited)
	}
}

func TestTraverseBudgetTruncates(t *testing.T) {
	cfg := defaultConfig()
	cfg.RunPageBudget = 1
	visitor := &fakeVisitor{queued: map[string][]pagevisit.VisitOutcome{
		"http://host/": {{
			Classification: pagevisit.Succeeded,
			DiscoveredURLs: []string{"http://host/a", "http://host/b"},
		}},
	}}
	observer := newObserver()
	traverser := ordertraversal.NewOrderTraverser(cfg, visitor, observer, &manualClock{})

	traverse(t, traverser, []string{"http://host/"})

	if observer.budget != 1 || observer.disposed[crawlcapability.DisposalBudgetTruncated] == 0 {
		t.Fatalf("budget not exhausted: budget=%d disposed=%v", observer.budget, observer.disposed)
	}
}

func TestTraverseDefersThenGivesUp(t *testing.T) {
	deferred := pagevisit.VisitOutcome{Classification: pagevisit.Deferred, DeferFor: time.Second}
	visitor := &fakeVisitor{queued: map[string][]pagevisit.VisitOutcome{
		"http://host/": {deferred, deferred, deferred, deferred},
	}}
	observer := newObserver()
	traverser := ordertraversal.NewOrderTraverser(
		defaultConfig(),
		visitor,
		observer,
		&manualClock{},
	)

	traverse(t, traverser, []string{"http://host/"})

	if observer.refusals[crawlcapability.RefusalDefer] == 0 {
		t.Fatal("expected defer refusals")
	}
	if observer.disposed[crawlcapability.DisposalFetchFailed] != 1 {
		t.Fatalf("expected fetch-failed after defer limit, got %v", observer.disposed)
	}
}

func TestTraverseRetriesTransientFetchThenSucceeds(t *testing.T) {
	transient := pagevisit.VisitOutcome{Classification: pagevisit.Transient}
	visitor := &fakeVisitor{queued: map[string][]pagevisit.VisitOutcome{
		"http://host/": {transient, transient, {Classification: pagevisit.Succeeded}},
	}}
	traverser := ordertraversal.NewOrderTraverser(
		defaultConfig(),
		visitor,
		newObserver(),
		&manualClock{},
	)

	traverse(t, traverser, []string{"http://host/"})

	if visitor.visitCount("http://host/") != 3 {
		t.Fatalf(
			"want 3 visits (2 transient + success), got %d",
			visitor.visitCount("http://host/"),
		)
	}
}

func TestTraverseAbandonsTransientFetchAfterLimit(t *testing.T) {
	transient := pagevisit.VisitOutcome{Classification: pagevisit.Transient}
	visitor := &fakeVisitor{queued: map[string][]pagevisit.VisitOutcome{
		"http://host/": {transient, transient, transient},
	}}
	observer := newObserver()
	traverser := ordertraversal.NewOrderTraverser(
		defaultConfig(),
		visitor,
		observer,
		&manualClock{},
	)

	traverse(t, traverser, []string{"http://host/"})

	if observer.disposed[crawlcapability.DisposalFetchFailed] != 1 {
		t.Fatalf("expected fetch-failed after retry limit, got %v", observer.disposed)
	}
}

func TestTraverseSkipsUncanonicalizableSeed(t *testing.T) {
	visitor := &fakeVisitor{queued: map[string][]pagevisit.VisitOutcome{}}
	traverser := ordertraversal.NewOrderTraverser(
		defaultConfig(),
		visitor,
		newObserver(),
		&manualClock{},
	)

	traverse(t, traverser, []string{"::not a url"})

	if len(visitor.visited) != 0 {
		t.Fatalf("uncanonicalizable seed should not be visited, got %v", visitor.visited)
	}
}

func TestTraverseVisitorErrorFails(t *testing.T) {
	visitor := &fakeVisitor{err: errors.New("boom")}
	traverser := ordertraversal.NewOrderTraverser(
		defaultConfig(),
		visitor,
		newObserver(),
		&manualClock{},
	)

	if err := traverser.Traverse(
		context.Background(),
		orderDelivery([]string{"http://host/"}),
	); err == nil {
		t.Fatal("visitor error should fail the traversal")
	}
}
