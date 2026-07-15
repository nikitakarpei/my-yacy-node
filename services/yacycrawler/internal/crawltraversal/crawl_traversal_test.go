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

type fakeExtract struct {
	documents []crawlcapability.ExtractedDocument
	err       error
}

func (f fakeExtract) Extract(
	_ context.Context,
	_, _ string,
	_ []byte,
) ([]crawlcapability.ExtractedDocument, error) {
	return f.documents, f.err
}

func document(url, title, text string) crawlcapability.ExtractedDocument {
	return crawlcapability.ExtractedDocument{
		URL: url,
		ExtractedContent: crawlcapability.ExtractedContent{
			Title:  title,
			Body:   []byte(text),
			Format: crawlcapability.PageContentFormatText,
		},
	}
}

type fakeRecrawl struct{ due bool }

func (f fakeRecrawl) Due(context.Context, string) (bool, error) { return f.due, nil }

type fakeOutput struct {
	name      string
	refuses   crawlcapability.PageContentFormat
	mu        sync.Mutex
	published []string
	failWith  error
}

func (o *fakeOutput) Name() string { return o.name }

func (o *fakeOutput) Accepts(format crawlcapability.PageContentFormat) bool {
	return format != o.refuses
}

func (o *fakeOutput) Derive(
	page crawlcapability.CrawledPage,
	_ *crawlcapability.RenderedContent,
) (crawlcapability.CrawledPage, error) {
	return page, nil
}

func (o *fakeOutput) Publish(_ context.Context, page crawlcapability.CrawledPage) error {
	if o.failWith != nil {
		return o.failWith
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.published = append(o.published, page.CanonicalURL)
	return nil
}

type identityOutput interface {
	crawlcapability.RepresentationDerivation[crawlcapability.CrawledPage]
	crawlcapability.PagePublication[crawlcapability.CrawledPage]
}

func outputs(items ...identityOutput) []crawlcapability.PageRepresentationOutput {
	bound := make([]crawlcapability.PageRepresentationOutput, len(items))
	for i, item := range items {
		bound[i] = crawlcapability.BindRepresentation(item, item)
	}
	return bound
}

type recordingObserver struct {
	mu        sync.Mutex
	disposed  map[string]int
	published map[string]int
	refusals  map[string]int
	budget    int
}

func (*recordingObserver) OrderReceived()              {}
func (*recordingObserver) OrderRedelivered()           {}
func (*recordingObserver) OrderCompleted()             {}
func (*recordingObserver) PageFetched()                {}
func (*recordingObserver) PublicationWaited()          {}
func (*recordingObserver) FetchObserved(time.Duration) {}

func newObserver() *recordingObserver {
	return &recordingObserver{
		disposed:  map[string]int{},
		published: map[string]int{},
		refusals:  map[string]int{},
	}
}

func (o *recordingObserver) PageDisposed(reason string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.disposed[reason]++
}

func (o *recordingObserver) PagePublished(out string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.published[out]++
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
	extract crawlcapability.DocumentExtraction,
	outputs []crawlcapability.PageRepresentationOutput,
	observer crawlcapability.RunProgress,
) *crawltraversal.Crawler {
	return crawltraversal.NewCrawler(
		cfg, fetch, extract, crawltraversal.AlwaysDue{},
		outputs, observer, &manualClock{},
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

func TestTraversePublishesToEveryOutput(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	extract := fakeExtract{
		documents: []crawlcapability.ExtractedDocument{document("http://host/", "t", "body")},
	}
	rwi := &fakeOutput{name: "rwi"}
	text := &fakeOutput{name: "text"}
	crawler := newCrawler(defaultConfig(), fetch, extract,
		outputs(rwi, text), newObserver())

	traverse(t, crawler, []string{"http://host/"})

	if len(rwi.published) != 1 || len(text.published) != 1 {
		t.Fatalf("representations not both advanced: rwi=%v text=%v", rwi.published, text.published)
	}
}

func TestTraverseSkipsRepresentationRefusingPageFormat(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	extract := fakeExtract{
		documents: []crawlcapability.ExtractedDocument{document("http://host/", "t", "body")},
	}
	rwi := &fakeOutput{name: "rwi"}
	markdown := &fakeOutput{name: "markdown", refuses: crawlcapability.PageContentFormatText}
	observer := newObserver()
	crawler := newCrawler(defaultConfig(), fetch, extract,
		outputs(rwi, markdown), observer)

	traverse(t, crawler, []string{"http://host/"})

	if len(rwi.published) != 1 {
		t.Fatalf("accepting representation not advanced: rwi=%v", rwi.published)
	}
	if len(markdown.published) != 0 {
		t.Fatalf("refusing representation advanced: markdown=%v", markdown.published)
	}
	if observer.disposed[crawlcapability.DisposalUnrepresentable] != 0 {
		t.Fatalf("page disposed despite an accepting representation: %v", observer.disposed)
	}
}

func TestTraverseDisposesPageNoRepresentationAccepts(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	extract := fakeExtract{
		documents: []crawlcapability.ExtractedDocument{document("http://host/", "t", "body")},
	}
	rwi := &fakeOutput{name: "rwi", refuses: crawlcapability.PageContentFormatText}
	observer := newObserver()
	crawler := newCrawler(defaultConfig(), fetch, extract,
		outputs(rwi), observer)

	traverse(t, crawler, []string{"http://host/"})

	if len(rwi.published) != 0 {
		t.Fatalf("refusing representation advanced: rwi=%v", rwi.published)
	}
	if observer.disposed[crawlcapability.DisposalUnrepresentable] != 1 {
		t.Fatalf("want unrepresentable disposal, got %v", observer.disposed)
	}
}

func TestTraverseDisposesUnsupportedMediaType(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	extract := fakeExtract{err: crawlcapability.ErrUnsupportedMediaType}
	observer := newObserver()
	crawler := newCrawler(defaultConfig(), fetch, extract,
		outputs(&fakeOutput{name: "rwi"}), observer)

	traverse(t, crawler, []string{"http://host/"})

	if observer.disposed[crawlcapability.DisposalUnsupportedMediaType] != 1 {
		t.Fatalf("want unsupported-media-type disposal, got %v", observer.disposed)
	}
}

func TestTraverseDisposesContainerOverflow(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	extract := fakeExtract{err: crawlcapability.ErrContainerOverflow}
	observer := newObserver()
	crawler := newCrawler(defaultConfig(), fetch, extract,
		outputs(&fakeOutput{name: "rwi"}), observer)

	traverse(t, crawler, []string{"http://host/"})

	if observer.disposed[crawlcapability.DisposalContainerOverflow] != 1 {
		t.Fatalf("want container-overflow disposal, got %v", observer.disposed)
	}
}

func TestTraverseDisposesEmptyExtraction(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	extract := fakeExtract{documents: nil}
	observer := newObserver()
	crawler := newCrawler(defaultConfig(), fetch, extract,
		outputs(&fakeOutput{name: "rwi"}), observer)

	traverse(t, crawler, []string{"http://host/"})

	if observer.disposed[crawlcapability.DisposalUnextractable] != 1 {
		t.Fatalf("want unextractable disposal, got %v", observer.disposed)
	}
}

func TestTraverseFansOutContainerDocuments(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/a.zip": {crawlcapability.FetchOutcome{
			Status: crawlcapability.FetchSucceeded, FinalURL: "http://host/a.zip",
			ContentType: "application/zip", Body: []byte("x"),
		}},
	}}
	extract := fakeExtract{documents: []crawlcapability.ExtractedDocument{
		document("http://host/a.zip!/one.html", "one", "a"),
		document("http://host/a.zip!/two.html", "two", "b"),
	}}
	rwi := &fakeOutput{name: "rwi"}
	crawler := newCrawler(defaultConfig(), fetch, extract,
		outputs(rwi), newObserver())

	traverse(t, crawler, []string{"http://host/a.zip"})

	if len(rwi.published) != 2 {
		t.Fatalf("want 2 member documents published, got %v", rwi.published)
	}
	if rwi.published[0] == rwi.published[1] {
		t.Fatalf("members collapsed to one URL: %v", rwi.published)
	}
}

func TestTraverseHonorsMetaNoIndex(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	extract := fakeExtract{
		documents: []crawlcapability.ExtractedDocument{{
			URL: "http://host/",
			ExtractedContent: crawlcapability.ExtractedContent{
				Body:            []byte("b"),
				Format:          crawlcapability.PageContentFormatText,
				RefusesIndexing: true,
			},
		}},
	}
	rwi := &fakeOutput{name: "rwi"}
	observer := newObserver()
	crawler := newCrawler(defaultConfig(), fetch, extract,
		outputs(rwi), observer)

	traverse(t, crawler, []string{"http://host/"})

	if len(rwi.published) != 0 || observer.disposed[crawlcapability.DisposalNoIndex] != 1 {
		t.Fatalf(
			"noindex not honored: published=%v disposed=%v",
			rwi.published,
			observer.disposed,
		)
	}
}

func TestTraverseHonorsNoFollow(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {crawlcapability.FetchOutcome{
			Status: crawlcapability.FetchSucceeded, FinalURL: "http://host/",
			ContentType: "text/html", Body: []byte("x"), RefusesLinkDiscovery: true,
		}},
	}}
	extract := fakeExtract{documents: []crawlcapability.ExtractedDocument{
		{
			URL: "http://host/",
			ExtractedContent: crawlcapability.ExtractedContent{
				Body:   []byte("b"),
				Format: crawlcapability.PageContentFormatText,
				Links:  []string{"http://host/next"},
			},
		},
	}}
	rwi := &fakeOutput{name: "rwi"}
	crawler := newCrawler(defaultConfig(), fetch, extract,
		outputs(rwi), newObserver())

	traverse(t, crawler, []string{"http://host/"})

	if len(rwi.published) != 1 {
		t.Fatalf("want only the seed published, got %v", rwi.published)
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
	callCount := 0
	extract := extractFunc(func() ([]crawlcapability.ExtractedDocument, error) {
		callCount++
		if callCount == 1 {
			return []crawlcapability.ExtractedDocument{
				{
					URL: "http://host/",
					ExtractedContent: crawlcapability.ExtractedContent{
						Body:   []byte("b"),
						Format: crawlcapability.PageContentFormatText,
						Links:  []string{"http://host/next"},
					},
				},
			}, nil
		}
		return []crawlcapability.ExtractedDocument{document("http://host/next", "", "c")}, nil
	})
	rwi := &fakeOutput{name: "rwi"}
	crawler := newCrawler(defaultConfig(), fetch, extract,
		outputs(rwi), newObserver())

	traverse(t, crawler, []string{"http://host/"})

	if len(rwi.published) != 2 {
		t.Fatalf("want seed plus discovered link, got %v", rwi.published)
	}
}

type extractFunc func() ([]crawlcapability.ExtractedDocument, error)

func (f extractFunc) Extract(
	_ context.Context,
	_, _ string,
	_ []byte,
) ([]crawlcapability.ExtractedDocument, error) {
	return f()
}

func TestTraverseSkipsFetchWhenNotDue(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	rwi := &fakeOutput{name: "rwi"}
	crawler := crawltraversal.NewCrawler(
		defaultConfig(), fetch, fakeExtract{}, fakeRecrawl{due: false},
		outputs(rwi), newObserver(), &manualClock{},
	)

	traverse(t, crawler, []string{"http://host/"})

	if len(rwi.published) != 0 {
		t.Fatalf("not-due seed should not be fetched or published, got %v", rwi.published)
	}
}

func TestTraverseRetriesTransientFetchThenSucceeds(t *testing.T) {
	transient := crawlcapability.FetchOutcome{Status: crawlcapability.FetchTransient}
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {transient, transient, fetchedOutcome()},
	}}
	extract := fakeExtract{
		documents: []crawlcapability.ExtractedDocument{document("http://host/", "", "b")},
	}
	rwi := &fakeOutput{name: "rwi"}
	crawler := newCrawler(defaultConfig(), fetch, extract,
		outputs(rwi), newObserver())

	traverse(t, crawler, []string{"http://host/"})

	if len(rwi.published) != 1 {
		t.Fatalf("transient fetch should retry then publish, got %v", rwi.published)
	}
}

func TestTraverseAbandonsTransientFetchAfterLimit(t *testing.T) {
	transient := crawlcapability.FetchOutcome{Status: crawlcapability.FetchTransient}
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {transient, transient, transient},
	}}
	rwi := &fakeOutput{name: "rwi"}
	observer := newObserver()
	crawler := newCrawler(defaultConfig(), fetch, fakeExtract{},
		outputs(rwi), observer)

	traverse(t, crawler, []string{"http://host/"})

	if len(rwi.published) != 0 {
		t.Fatalf("abandoned fetch should not publish, got %v", rwi.published)
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
	extract := fakeExtract{
		documents: []crawlcapability.ExtractedDocument{document("http://host/", "", "b")},
	}
	rwi := &fakeOutput{name: "rwi"}
	crawler := newCrawler(cfg, fetch, extract,
		outputs(rwi), newObserver())

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
	if len(rwi.published) != 1 {
		t.Fatalf("gated fetch should publish once heartbeat opens it, got %v", rwi.published)
	}
}

func TestTraverseCeasesOnHTTPCease(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {crawlcapability.FetchOutcome{Status: crawlcapability.FetchCeased}},
	}}
	observer := newObserver()
	crawler := newCrawler(defaultConfig(), fetch, fakeExtract{},
		outputs(&fakeOutput{name: "rwi"}), observer)

	traverse(t, crawler, []string{"http://host/"})

	if observer.refusals[crawlcapability.RefusalCeased] != 1 {
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
	crawler := newCrawler(defaultConfig(), fetch, fakeExtract{},
		outputs(&fakeOutput{name: "rwi"}), observer)

	traverse(t, crawler, []string{"http://host/"})

	if observer.refusals[crawlcapability.RefusalDeferred] == 0 {
		t.Fatal("expected defer refusals")
	}
	if observer.disposed[crawlcapability.DisposalFetchFailed] != 1 {
		t.Fatalf("expected fetch-failed after defer limit, got %v", observer.disposed)
	}
}

func TestTraverseDisposesOversized(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {crawlcapability.FetchOutcome{
			Status: crawlcapability.FetchSucceeded, FinalURL: "http://host/", Truncated: true,
		}},
	}}
	observer := newObserver()
	crawler := newCrawler(defaultConfig(), fetch, fakeExtract{},
		outputs(&fakeOutput{name: "rwi"}), observer)

	traverse(t, crawler, []string{"http://host/"})

	if observer.disposed[crawlcapability.DisposalOversized] != 1 {
		t.Fatalf("want oversized disposal, got %v", observer.disposed)
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
	extract := fakeExtract{documents: []crawlcapability.ExtractedDocument{
		{
			URL: "http://host/",
			ExtractedContent: crawlcapability.ExtractedContent{
				Body:   []byte("b"),
				Format: crawlcapability.PageContentFormatText,
				Links:  []string{"http://host/a", "http://host/b"},
			},
		},
	}}
	observer := newObserver()
	crawler := newCrawler(cfg, fetch, extract,
		outputs(&fakeOutput{name: "rwi"}), observer)

	traverse(t, crawler, []string{"http://host/"})

	if observer.budget != 1 || observer.disposed[crawlcapability.DisposalBudgetTruncated] == 0 {
		t.Fatalf("budget not exhausted: budget=%d disposed=%v", observer.budget, observer.disposed)
	}
}

func TestTraversePublicationHardErrorFails(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	extract := fakeExtract{
		documents: []crawlcapability.ExtractedDocument{document("http://host/", "", "b")},
	}
	output := &fakeOutput{name: "rwi", failWith: errors.New("hard broker error")}
	crawler := newCrawler(defaultConfig(), fetch, extract,
		outputs(output), newObserver())

	if err := crawler.Traverse(
		context.Background(),
		orderDelivery([]string{"http://host/"}),
	); err == nil {
		t.Fatal("hard publish error should fail the traversal")
	}
}

func TestTraverseRetriesTransientPublication(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{
		"http://host/": {fetchedOutcome()},
	}}
	extract := fakeExtract{
		documents: []crawlcapability.ExtractedDocument{document("http://host/", "", "b")},
	}
	output := &flakyOutput{failuresLeft: 2}
	crawler := newCrawler(defaultConfig(), fetch, extract,
		outputs(output), newObserver())

	traverse(t, crawler, []string{"http://host/"})

	if output.published != 1 {
		t.Fatalf("transient publish should retry then succeed: published=%d", output.published)
	}
}

type flakyOutput struct {
	failuresLeft int
	published    int
}

func (o *flakyOutput) Name() string { return "rwi" }

func (o *flakyOutput) Accepts(crawlcapability.PageContentFormat) bool { return true }

func (o *flakyOutput) Derive(
	page crawlcapability.CrawledPage,
	_ *crawlcapability.RenderedContent,
) (crawlcapability.CrawledPage, error) {
	return page, nil
}

func (o *flakyOutput) Publish(context.Context, crawlcapability.CrawledPage) error {
	if o.failuresLeft > 0 {
		o.failuresLeft--
		return crawlcapability.TransientPublicationError{Err: errors.New("stream full")}
	}
	o.published++
	return nil
}

func TestTraverseFetchErrorFails(t *testing.T) {
	fetch := &fakeFetch{err: errors.New("boom")}
	crawler := newCrawler(defaultConfig(), fetch, fakeExtract{},
		outputs(&fakeOutput{name: "rwi"}), newObserver())

	if err := crawler.Traverse(
		context.Background(),
		orderDelivery([]string{"http://host/"}),
	); err == nil {
		t.Fatal("fetch error should fail the traversal")
	}
}

func TestTraverseSkipsUncanonicalizableSeed(t *testing.T) {
	fetch := &fakeFetch{outcomes: map[string][]crawlcapability.FetchOutcome{}}
	crawler := newCrawler(defaultConfig(), fetch, fakeExtract{},
		outputs(&fakeOutput{name: "rwi"}), newObserver())

	traverse(t, crawler, []string{"::not a url"})
}
