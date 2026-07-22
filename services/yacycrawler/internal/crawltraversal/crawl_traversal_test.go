package crawltraversal_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawltraversal"
)

type fakeFetch struct {
	mu       sync.Mutex
	outcomes map[string][]crawlcapability.FetchOutcome
	err      error
}

func (f *fakeFetch) Fetch(_ context.Context, rawURL string) (crawlcapability.FetchOutcome, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return crawlcapability.FetchOutcome{}, f.err
	}
	queue := f.outcomes[rawURL]
	if len(queue) == 0 {
		return crawlcapability.FetchOutcome{Status: crawlcapability.FetchNotAPage}, nil
	}
	outcome := queue[0]
	if len(queue) > 1 {
		f.outcomes[rawURL] = queue[1:]
	}
	return outcome, nil
}

type fakeRecrawl struct{ due bool }

func (f fakeRecrawl) Due(context.Context, string) (bool, error) { return f.due, nil }

type fakeAbsorption struct {
	mu       sync.Mutex
	links    map[string][]string
	absorbed []string
	err      error
}

func (a *fakeAbsorption) Absorb(
	_ context.Context,
	outcome crawlcapability.FetchOutcome,
) ([]string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.err != nil {
		return nil, a.err
	}
	a.absorbed = append(a.absorbed, outcome.FinalURL)
	return a.links[outcome.FinalURL], nil
}

func (a *fakeAbsorption) count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.absorbed)
}

type recordingObserver struct {
	mu       sync.Mutex
	disposed map[string]int
	refusals map[string]int
	budget   int
}

func (*recordingObserver) OrderReceived()              {}
func (*recordingObserver) OrderRedelivered()           {}
func (*recordingObserver) OrderCompleted()             {}
func (*recordingObserver) PageFetched()                {}
func (*recordingObserver) PagePublished(string)        {}
func (*recordingObserver) PublicationWaited()          {}
func (*recordingObserver) FetchObserved(time.Duration) {}

func newObserver() *recordingObserver {
	return &recordingObserver{
		disposed: map[string]int{},
		refusals: map[string]int{},
	}
}

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

func defaultConfig() crawltraversal.Config {
	return crawltraversal.Config{
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

func newCrawler(
	cfg crawltraversal.Config,
	fetch crawlcapability.PageRetrieval,
	absorption crawlcapability.PageAbsorption,
	observer crawlcapability.RunProgress,
) *crawltraversal.Crawler {
	return crawltraversal.NewCrawler(
		cfg, fetch, crawltraversal.AlwaysDue{}, absorption, observer, &manualClock{},
	)
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

func traverse(t *testing.T, crawler *crawltraversal.Crawler, seeds []string) {
	t.Helper()
	if err := crawler.Traverse(context.Background(), orderDelivery(seeds)); err != nil {
		t.Fatalf("traverse: %v", err)
	}
}

func fetchedOutcome() crawlcapability.FetchOutcome {
	return crawlcapability.FetchOutcome{
		Status: crawlcapability.FetchSucceeded, FinalURL: "http://host/", ContentType: "text/html",
		Body: []byte("x"),
	}
}

func TestTraverseAbsorbsFetchedPage(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	absorption := &fakeAbsorption{}
	crawler := newCrawler(defaultConfig(), fetch, absorption, newObserver())

	traverse(t, crawler, []string{"http://host/"})

	if absorption.count() != 1 {
		t.Fatalf("fetched page not absorbed: %v", absorption.absorbed)
	}
}

func TestTraverseAbsorptionErrorFails(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	absorption := &fakeAbsorption{err: errors.New("absorb boom")}
	crawler := newCrawler(defaultConfig(), fetch, absorption, newObserver())

	if err := crawler.Traverse(
		context.Background(),
		orderDelivery([]string{"http://host/"}),
	); err == nil {
		t.Fatal("absorption error should fail the traversal")
	}
}

func TestTraverseDisposesNotAPage(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {crawlcapability.FetchOutcome{Status: crawlcapability.FetchNotAPage}},
	}}
	observer := newObserver()
	crawler := newCrawler(defaultConfig(), fetch, &fakeAbsorption{}, observer)

	traverse(t, crawler, []string{"http://host/"})

	if observer.disposed[crawlcapability.DisposalFetchFailed] != 1 {
		t.Fatalf("want fetch-failed disposal for non-page, got %v", observer.disposed)
	}
}

func TestTraverseDiscoversAndCrawlsLinks(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {crawlcapability.FetchOutcome{
			Status: crawlcapability.FetchSucceeded, FinalURL: "http://host/",
			ContentType: "text/html", Body: []byte("x"),
		}},
		"http://host/next": {crawlcapability.FetchOutcome{
			Status: crawlcapability.FetchSucceeded, FinalURL: "http://host/next",
			ContentType: "text/html", Body: []byte("y"),
		}},
	}}
	absorption := &fakeAbsorption{links: map[string][]string{
		"http://host/": {"http://host/next"},
	}}
	crawler := newCrawler(defaultConfig(), fetch, absorption, newObserver())

	traverse(t, crawler, []string{"http://host/"})

	if absorption.count() != 2 {
		t.Fatalf("want seed plus discovered link absorbed, got %v", absorption.absorbed)
	}
}

func TestTraverseSkipsFetchWhenNotDue(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	absorption := &fakeAbsorption{}
	crawler := crawltraversal.NewCrawler(
		defaultConfig(), fetch, fakeRecrawl{due: false}, absorption, newObserver(), &manualClock{},
	)

	traverse(t, crawler, []string{"http://host/"})

	if absorption.count() != 0 {
		t.Fatalf("not-due seed should not be fetched or absorbed, got %v", absorption.absorbed)
	}
}

func TestTraverseRetriesTransientFetchThenSucceeds(t *testing.T) {
	transient := crawlcapability.FetchOutcome{Status: crawlcapability.FetchTransient}
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {transient, transient, fetchedOutcome()},
	}}
	absorption := &fakeAbsorption{}
	crawler := newCrawler(defaultConfig(), fetch, absorption, newObserver())

	traverse(t, crawler, []string{"http://host/"})

	if absorption.count() != 1 {
		t.Fatalf("transient fetch should retry then absorb, got %v", absorption.absorbed)
	}
}

func TestTraverseAbandonsTransientFetchAfterLimit(t *testing.T) {
	transient := crawlcapability.FetchOutcome{Status: crawlcapability.FetchTransient}
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {transient, transient, transient},
	}}
	observer := newObserver()
	absorption := &fakeAbsorption{}
	crawler := newCrawler(defaultConfig(), fetch, absorption, observer)

	traverse(t, crawler, []string{"http://host/"})

	if absorption.count() != 0 {
		t.Fatalf("abandoned fetch should not absorb, got %v", absorption.absorbed)
	}
	if observer.disposed[crawlcapability.DisposalFetchFailed] != 1 {
		t.Fatalf("expected fetch-failed after retry limit, got %v", observer.disposed)
	}
}

type gatedFetch struct {
	gate    <-chan struct{}
	outcome crawlcapability.FetchOutcome
}

func (g gatedFetch) Fetch(ctx context.Context, _ string) (crawlcapability.FetchOutcome, error) {
	select {
	case <-g.gate:
		return g.outcome, nil
	case <-ctx.Done():
		return crawlcapability.FetchOutcome{}, fmt.Errorf("gated fetch: %w", ctx.Err())
	}
}

func TestTraverseRenewsOwnershipWhileCrawling(t *testing.T) {
	cfg := defaultConfig()
	cfg.OwnershipInterval = time.Millisecond

	gate := make(chan struct{})
	var openOnce sync.Once
	var renewed atomic.Int64
	fetch := gatedFetch{gate: gate, outcome: fetchedOutcome()}
	absorption := &fakeAbsorption{}
	crawler := newCrawler(cfg, fetch, absorption, newObserver())

	delivery := crawlcapability.DeliveredOrder{
		Order: yacycrawlcontract.CrawlOrder{
			OrderID: "o1", Profile: wideProfile(), SeedURLs: []string{"http://host/"},
		},
		Ack:   func(context.Context) error { return nil },
		Retry: func(context.Context) error { return nil },
		ExtendOwnership: func(context.Context) error {
			renewed.Add(1)
			openOnce.Do(func() { close(gate) })
			return nil
		},
	}

	if err := crawler.Traverse(context.Background(), delivery); err != nil {
		t.Fatalf("traverse: %v", err)
	}
	if renewed.Load() == 0 {
		t.Fatal("expected ownership heartbeat to extend at least once")
	}
	if absorption.count() != 1 {
		t.Fatalf("gated fetch should absorb once heartbeat opens it, got %v", absorption.absorbed)
	}
}

func TestTraverseCeasesOnHTTPCease(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {crawlcapability.FetchOutcome{Status: crawlcapability.FetchCeased}},
	}}
	observer := newObserver()
	crawler := newCrawler(defaultConfig(), fetch, &fakeAbsorption{}, observer)

	traverse(t, crawler, []string{"http://host/"})

	if observer.refusals[crawlcapability.RefusalCease] != 1 {
		t.Fatalf("cease not honored: %v", observer.refusals)
	}
}

func TestTraverseDefersThenGivesUp(t *testing.T) {
	deferred := crawlcapability.FetchOutcome{
		Status:   crawlcapability.FetchDeferred,
		DeferFor: time.Second,
	}
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {deferred, deferred, deferred, deferred},
	}}
	observer := newObserver()
	crawler := newCrawler(defaultConfig(), fetch, &fakeAbsorption{}, observer)

	traverse(t, crawler, []string{"http://host/"})

	if observer.refusals[crawlcapability.RefusalDefer] == 0 {
		t.Fatal("expected defer refusals")
	}
	if observer.disposed[crawlcapability.DisposalFetchFailed] != 1 {
		t.Fatalf("expected fetch-failed after defer limit, got %v", observer.disposed)
	}
}

func TestTraverseBudgetTruncates(t *testing.T) {
	cfg := defaultConfig()
	cfg.RunPageBudget = 1
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {crawlcapability.FetchOutcome{
			Status: crawlcapability.FetchSucceeded, FinalURL: "http://host/",
			ContentType: "text/html", Body: []byte("x"),
		}},
	}}
	absorption := &fakeAbsorption{links: map[string][]string{
		"http://host/": {"http://host/a", "http://host/b"},
	}}
	observer := newObserver()
	crawler := newCrawler(cfg, fetch, absorption, observer)

	traverse(t, crawler, []string{"http://host/"})

	if observer.budget != 1 || observer.disposed[crawlcapability.DisposalBudgetTruncated] == 0 {
		t.Fatalf("budget not exhausted: budget=%d disposed=%v", observer.budget, observer.disposed)
	}
}

func TestTraverseFetchErrorFails(t *testing.T) {
	fetch := &fakeFetch{err: errors.New("boom")}
	crawler := newCrawler(defaultConfig(), fetch, &fakeAbsorption{}, newObserver())

	if err := crawler.Traverse(
		context.Background(),
		orderDelivery([]string{"http://host/"}),
	); err == nil {
		t.Fatal("fetch error should fail the traversal")
	}
}

func TestTraverseSkipsUncanonicalizableSeed(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{}}
	crawler := newCrawler(defaultConfig(), fetch, &fakeAbsorption{}, newObserver())

	traverse(t, crawler, []string{"::not a url"})
}
