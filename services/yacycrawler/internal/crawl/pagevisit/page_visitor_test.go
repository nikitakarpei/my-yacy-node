package pagevisit_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkdiscovery"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

type fakeFetch struct {
	mu         sync.Mutex
	outcome    pagefetch.FetchOutcome
	err        error
	gotVersion pagefetch.PageVersion
	sawFetch   bool
}

func (f *fakeFetch) Fetch(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	knownVersion pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gotVersion = knownVersion
	f.sawFetch = true
	if f.err != nil {
		return pagefetch.FetchOutcome{}, f.err
	}
	return f.outcome, nil
}

type visitedCall struct {
	url     canonicalurl.CanonicalURL
	version pagefetch.PageVersion
}

type fakePageVisits struct {
	mu      sync.Mutex
	due     bool
	version pagefetch.PageVersion

	visitedCalls []visitedCall
}

func (f *fakePageVisits) LastPageVisitOf(
	context.Context,
	canonicalurl.CanonicalURL,
) (pagevisit.LastPageVisit, bool) {
	return pagevisit.LastPageVisit{Version: f.version}, true
}

func (f *fakePageVisits) PageDueForRecrawl(pagevisit.LastPageVisit) bool {
	return f.due
}

func (f *fakePageVisits) RecordPageVisit(
	_ context.Context,
	url canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.visitedCalls = append(
		f.visitedCalls,
		visitedCall{url: url, version: version},
	)
}

func (f *fakePageVisits) calls() []visitedCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]visitedCall(nil), f.visitedCalls...)
}

type steppingClock struct {
	now  time.Time
	step time.Duration
}

func (c *steppingClock) Now() time.Time {
	current := c.now
	c.now = c.now.Add(c.step)
	return current
}

type recordingObserver struct {
	mu                            sync.Mutex
	fetchDurations                []time.Duration
	fetched                       int
	fetchesCanceled               int
	linkDiscoveryRefusalsEnforced int
}

func newObserver() *recordingObserver {
	return &recordingObserver{}
}

func (o *recordingObserver) PageFetchSucceeded(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.fetchDurations = append(o.fetchDurations, duration)
	o.fetched++
}

func (o *recordingObserver) PageFetchNotModified(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
) {
	o.recordFetchDuration(duration)
}

func (o *recordingObserver) PageFetchAccessRefused(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
) {
	o.recordFetchDuration(duration)
}

func (o *recordingObserver) PageFetchDeferred(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
	_ time.Duration,
) {
	o.recordFetchDuration(duration)
}

func (o *recordingObserver) PageFetchRejected(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
) {
	o.recordFetchDuration(duration)
}

func (o *recordingObserver) PageFetchRedirected(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
) {
	o.recordFetchDuration(duration)
}

func (o *recordingObserver) PageFetchRedirectTargetInvalid(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
	_ error,
) {
	o.recordFetchDuration(duration)
}

func (o *recordingObserver) PageFetchRefusedOversizedPage(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
) {
	o.recordFetchDuration(duration)
}

func (o *recordingObserver) PageFetchFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
	_ error,
) {
	o.recordFetchDuration(duration)
}

func (o *recordingObserver) recordFetchDuration(duration time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.fetchDurations = append(o.fetchDurations, duration)
}

func (o *recordingObserver) LinkDiscoveryRefusalEnforced(
	context.Context,
	canonicalurl.CanonicalURL,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.linkDiscoveryRefusalsEnforced++
}

func (o *recordingObserver) PageFetchCanceled(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.fetchDurations = append(o.fetchDurations, duration)
	o.fetchesCanceled++
}

type fakeCrawledPages struct {
	mu        sync.Mutex
	indexable []string
	refused   []string
}

func (f *fakeCrawledPages) PublishIndexablePage(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.indexable = append(f.indexable, pageURL.String())
}

func (f *fakeCrawledPages) PublishIndexingRefusedPage(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refused = append(f.refused, pageURL.String())
}

func (f *fakeCrawledPages) indexablePages() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.indexable...)
}

func (f *fakeCrawledPages) refusedPages() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.refused...)
}

func fetchedOutcome(t *testing.T) pagefetch.FetchOutcome {
	t.Helper()
	return pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			ContentType: "text/html",
			Body:        []byte(pageLinkingNext),
		},
		Version: pagefetch.PageVersion{EntityTag: `"etag"`},
	}
}

func unreadableOutcome(t *testing.T) pagefetch.FetchOutcome {
	t.Helper()
	outcome := fetchedOutcome(t)
	outcome.Page.ContentType = "application/pdf"
	return outcome
}

const fetchStep = 250 * time.Millisecond

func fetchOf(outcome pagefetch.FetchOutcome) *fakeFetch {
	return &fakeFetch{outcome: outcome}
}

func newPageVisitor(
	fetcher pagefetch.Fetcher,
	pageVisits *fakePageVisits,
	observer *recordingObserver,
	crawledPages pagevisit.CrawledPages,
) pagevisit.PageVisitor {
	pageFetcher := pagevisit.NewObservedPageFetcher(
		fetcher, &steppingClock{now: time.Unix(0, 0), step: fetchStep}, observer,
	)
	htmlPageReading := pagehtmlreading.NewHTMLPageReading(
		pagehtml.NewHTMLParser(silentMediaTypeObserver{}),
		linkdiscovery.NewLinkDiscovery(silentLinkResolutionObserver{}),
	)
	return pagevisit.New(
		pageFetcher, pageVisits, pageVisits, htmlPageReading, observer, crawledPages,
	)
}

func visitHostPage(t *testing.T, pageVisitor pagevisit.PageVisitor) pagevisit.PageVisitOutcome {
	t.Helper()
	outcome := pageVisitor.VisitPage(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)
	if outcome.Conclusion != pagevisit.PageVisitTerminal {
		t.Fatalf("want concluded, got %v", outcome.Conclusion)
	}
	return outcome
}

func TestVisitReadsTheFetchedPage(t *testing.T) {
	observer := newObserver()
	crawledPages := &fakeCrawledPages{}
	pageVisitor := newPageVisitor(
		fetchOf(fetchedOutcome(t)),
		&fakePageVisits{due: true},
		observer,
		crawledPages,
	)

	outcome := visitHostPage(t, pageVisitor)

	if observer.fetched != 1 {
		t.Fatalf("want one fetched page observed, got %d", observer.fetched)
	}

	if len(outcome.DiscoveredURLs) != 1 ||
		outcome.DiscoveredURLs[0] != canonicalurltest.CanonicalURLOf(t, "http://host/next") {
		t.Fatalf("want discovered link, got %v", outcome.DiscoveredURLs)
	}
	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("published page should report no disposal, got %q", outcome.Disposal)
	}
	if calls := crawledPages.indexablePages(); len(calls) != 1 || calls[0] != "http://host/" {
		t.Fatalf("want the page published as indexable once with its canonical url, got %v", calls)
	}
}

func TestVisitReportsFetchRejectedDisposal(t *testing.T) {
	recrawl := &fakePageVisits{due: true}
	crawledPages := &fakeCrawledPages{}
	pageVisitor := newPageVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchRejected}),
		recrawl,
		newObserver(),
		crawledPages,
	)

	outcome := visitHostPage(t, pageVisitor)

	if outcome.Disposal != disposal.FetchRejected {
		t.Fatalf("want fetch-rejected disposal, got %q", outcome.Disposal)
	}
	if len(recrawl.calls()) != 0 {
		t.Fatalf("visited should not be recorded for a rejected fetch, got %v", recrawl.calls())
	}
	if calls := crawledPages.indexablePages(); len(calls) != 0 {
		t.Fatalf("a rejected fetch should publish no crawled page, got %v", calls)
	}
}

func TestVisitReportsOversizedDisposal(t *testing.T) {
	crawledPages := &fakeCrawledPages{}
	pageVisitor := newPageVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchOversized}),
		&fakePageVisits{due: true},
		newObserver(),
		crawledPages,
	)

	outcome := visitHostPage(t, pageVisitor)

	if outcome.Disposal != disposal.Oversized {
		t.Fatalf("want oversized disposal, got %q", outcome.Disposal)
	}
	if calls := crawledPages.indexablePages(); len(calls) != 0 {
		t.Fatalf("an oversized page should publish no crawled page, got %v", calls)
	}
}

func TestVisitReportsRedirectTargetInvalidDisposal(t *testing.T) {
	crawledPages := &fakeCrawledPages{}
	pageVisitor := newPageVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchRedirectTargetInvalid}),
		&fakePageVisits{due: true},
		newObserver(),
		crawledPages,
	)

	outcome := visitHostPage(t, pageVisitor)

	if outcome.Disposal != disposal.RedirectTargetInvalid {
		t.Fatalf("want redirect-target-invalid disposal, got %q", outcome.Disposal)
	}
	if calls := crawledPages.indexablePages(); len(calls) != 0 {
		t.Fatalf("an invalid redirect target should publish no crawled page, got %v", calls)
	}
}

func TestVisitReportsTheRedirectTargetTheOriginAnsweredWith(t *testing.T) {
	crawledPages := &fakeCrawledPages{}
	redirectTarget := canonicalurltest.CanonicalURLOf(t, "http://host/moved")
	pageVisitor := newPageVisitor(
		fetchOf(pagefetch.FetchOutcome{
			Status:         pagefetch.FetchRedirected,
			RedirectTarget: redirectTarget,
		}),
		&fakePageVisits{due: true},
		newObserver(),
		crawledPages,
	)

	outcome := visitHostPage(t, pageVisitor)

	if outcome.RedirectTarget != redirectTarget {
		t.Fatalf("want redirect target %q, got %q", redirectTarget, outcome.RedirectTarget)
	}
	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("a redirect carries no disposal reason, got %q", outcome.Disposal)
	}
	if calls := crawledPages.indexablePages(); len(calls) != 0 {
		t.Fatalf("a redirect should publish no crawled page, got %v", calls)
	}
}

func TestVisitStopsWhenTheTargetRefusesAccess(t *testing.T) {
	observer := newObserver()
	recrawl := &fakePageVisits{due: true}
	crawledPages := &fakeCrawledPages{}
	pageVisitor := newPageVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchAccessRefused}),
		recrawl,
		observer,
		crawledPages,
	)

	outcome := visitHostPage(t, pageVisitor)

	if outcome.Disposal != disposal.AccessRefused {
		t.Fatalf("want access-refused disposal, got %q", outcome.Disposal)
	}
	if len(recrawl.calls()) != 1 {
		t.Fatalf(
			"visited should be recorded on a refusal so grace applies, got %v",
			recrawl.calls(),
		)
	}
	if calls := crawledPages.indexablePages(); len(calls) != 0 {
		t.Fatalf("a refused fetch should publish no crawled page, got %v", calls)
	}
}

func TestVisitReportsTransient(t *testing.T) {
	recrawl := &fakePageVisits{due: true}
	pageVisitor := newPageVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchFailed}),
		recrawl,
		newObserver(),
		&fakeCrawledPages{},
	)

	outcome := pageVisitor.VisitPage(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)
	if outcome.Conclusion != pagevisit.PageVisitRetryable {
		t.Fatalf("want retryable, got %v", outcome.Conclusion)
	}
	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("a transient failure must not dispose, got %q", outcome.Disposal)
	}
	if len(recrawl.calls()) != 0 {
		t.Fatalf("visited should not be recorded after failure, got %v", recrawl.calls())
	}
}

func TestAnUnreadableFetchStatusLeavesThePageForAnotherAttempt(t *testing.T) {
	pageVisitor := newPageVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchStatus(99)}),
		&fakePageVisits{due: true},
		newObserver(),
		&fakeCrawledPages{},
	)

	outcome := pageVisitor.VisitPage(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)

	if outcome.Conclusion != pagevisit.PageVisitRetryable {
		t.Fatalf("conclusion = %v, want retryable", outcome.Conclusion)
	}
}

func TestVisitReportsDeferred(t *testing.T) {
	pageVisitor := newPageVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchDeferred, DeferFor: time.Second}),
		&fakePageVisits{due: true},
		newObserver(),
		&fakeCrawledPages{},
	)

	outcome := pageVisitor.VisitPage(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)
	if outcome.Conclusion != pagevisit.PageVisitDeferred || outcome.DeferFor != time.Second {
		t.Fatalf("want deferred for 1s, got %+v", outcome)
	}
}

func TestVisitFetchErrorLeavesTheVisitRetryable(t *testing.T) {
	pageVisitor := newPageVisitor(
		&fakeFetch{err: errors.New("boom")},
		&fakePageVisits{due: true},
		newObserver(),
		&fakeCrawledPages{},
	)

	outcome := pageVisitor.VisitPage(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)
	if outcome.Conclusion != pagevisit.PageVisitRetryable {
		t.Fatalf("want the visit retryable, got %v", outcome.Conclusion)
	}
}

func TestAFetchThatIsCanceledIsObservedAsCanceled(t *testing.T) {
	observer := newObserver()
	pageVisitor := newPageVisitor(
		&fakeFetch{err: errors.New("boom")},
		&fakePageVisits{due: true},
		observer,
		&fakeCrawledPages{},
	)

	ctx, cancelVisit := context.WithCancel(context.Background())
	cancelVisit()
	pageVisitor.VisitPage(
		ctx,
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)

	if observer.fetchesCanceled != 1 {
		t.Fatalf("canceled fetches = %d, want 1", observer.fetchesCanceled)
	}
}

func TestVisitReportsHowLongTheFetchTook(t *testing.T) {
	observer := newObserver()

	visitHostPage(t, newPageVisitor(
		fetchOf(fetchedOutcome(t)),
		&fakePageVisits{due: true},
		observer,
		&fakeCrawledPages{},
	))

	if len(observer.fetchDurations) != 1 || observer.fetchDurations[0] != fetchStep {
		t.Fatalf("want one %v fetch observed, got %v", fetchStep, observer.fetchDurations)
	}
}

func TestVisitReportsHowLongAFailedFetchTook(t *testing.T) {
	observer := newObserver()
	pageVisitor := newPageVisitor(
		&fakeFetch{err: errors.New("boom")},
		&fakePageVisits{due: true},
		observer,
		&fakeCrawledPages{},
	)

	pageVisitor.VisitPage(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)

	if len(observer.fetchDurations) != 1 || observer.fetchDurations[0] != fetchStep {
		t.Fatalf("want one %v fetch observed, got %v", fetchStep, observer.fetchDurations)
	}
}

func TestVisitReportsNoFetchDurationWhenNotDue(t *testing.T) {
	observer := newObserver()

	visitHostPage(t, newPageVisitor(
		fetchOf(fetchedOutcome(t)),
		&fakePageVisits{due: false},
		observer,
		&fakeCrawledPages{},
	))

	if len(observer.fetchDurations) != 0 {
		t.Fatalf("want no fetch observed, got %v", observer.fetchDurations)
	}
}

type unreadableHTMLPage struct{ err error }

func (page unreadableHTMLPage) ReadingOfPage(
	context.Context,
	canonicalurl.CanonicalURL,
	pagefetch.FetchedPage,
) (pagehtmlreading.Reading, error) {
	return pagehtmlreading.Reading{}, page.err
}

func TestUnreadablePageHTMLIsDisposed(t *testing.T) {
	observer := newObserver()
	pageVisitor := pagevisit.New(
		pagevisit.NewObservedPageFetcher(
			fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchSucceeded}),
			&steppingClock{now: time.Unix(0, 0), step: fetchStep},
			observer,
		),
		&fakePageVisits{due: true},
		&fakePageVisits{due: true},
		unreadableHTMLPage{err: errors.New("boom")},
		observer,
		&fakeCrawledPages{},
	)

	outcome := visitHostPage(t, pageVisitor)

	if outcome.Disposal != disposal.UnreadableHTML {
		t.Fatalf("disposal = %q, want %q", outcome.Disposal, disposal.UnreadableHTML)
	}
}

func TestVisitReportsNotDueWithoutFetching(t *testing.T) {
	fetch := fetchOf(fetchedOutcome(t))
	pageVisitor := newPageVisitor(
		fetch,
		&fakePageVisits{due: false},
		newObserver(),
		&fakeCrawledPages{},
	)

	outcome := visitHostPage(t, pageVisitor)

	if outcome.Disposal != disposal.NotDue {
		t.Fatalf("want a not-due disposal, got %q", outcome.Disposal)
	}
	if fetch.sawFetch {
		t.Fatal("fetch should not be attempted when not due")
	}
}

func TestVisitPassesKnownVersionToFetcher(t *testing.T) {
	fetch := fetchOf(fetchedOutcome(t))
	known := pagefetch.PageVersion{EntityTag: `"stored-etag"`}
	pageVisitor := newPageVisitor(
		fetch,
		&fakePageVisits{due: true, version: known},
		newObserver(),
		&fakeCrawledPages{},
	)

	pageVisitor.VisitPage(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)
	if fetch.gotVersion != known {
		t.Fatalf("fetcher version = %+v, want %+v", fetch.gotVersion, known)
	}
}

func TestVisitRecordsVersionAfterReadingThePage(t *testing.T) {
	recrawl := &fakePageVisits{due: true}
	pageVisitor := newPageVisitor(
		fetchOf(fetchedOutcome(t)),
		recrawl,
		newObserver(),
		&fakeCrawledPages{},
	)

	pageVisitor.VisitPage(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)
	calls := recrawl.calls()
	if len(calls) != 1 {
		t.Fatalf("want exactly one visited call, got %v", calls)
	}
	if calls[0].url.String() != "http://host/" || calls[0].version.EntityTag != `"etag"` {
		t.Fatalf("visited call = %+v", calls[0])
	}
}

func TestVisitReportsTheDisposalAbsorptionReached(t *testing.T) {
	recrawl := &fakePageVisits{due: true}
	crawledPages := &fakeCrawledPages{}
	pageVisitor := newPageVisitor(
		fetchOf(unreadableOutcome(t)),
		recrawl,
		newObserver(),
		crawledPages,
	)

	outcome := visitHostPage(t, pageVisitor)

	if outcome.Disposal != disposal.UnsupportedMediaType {
		t.Fatalf("want the page content reason reported, got %q", outcome.Disposal)
	}
	if len(recrawl.calls()) != 1 {
		t.Fatalf("want the visit recorded regardless of the publication, got %v", recrawl.calls())
	}
	if calls := crawledPages.indexablePages(); len(calls) != 0 {
		t.Fatalf("a disposed page should publish no crawled page, got %v", calls)
	}
}

func TestVisitNotModifiedRecordsVersionWithoutReadingThePage(t *testing.T) {
	recrawl := &fakePageVisits{due: true}
	crawledPages := &fakeCrawledPages{}
	pageVisitor := newPageVisitor(
		fetchOf(
			pagefetch.FetchOutcome{
				Status:  pagefetch.FetchNotModified,
				Version: pagefetch.PageVersion{EntityTag: `"same"`},
			},
		),
		recrawl,
		newObserver(),
		crawledPages,
	)

	outcome := visitHostPage(t, pageVisitor)

	if outcome.Disposal != disposal.NotModified {
		t.Fatalf("want not-modified disposal, got %q", outcome.Disposal)
	}
	calls := recrawl.calls()
	if len(calls) != 1 || calls[0].version.EntityTag != `"same"` {
		t.Fatalf("want the version recorded once, got %v", calls)
	}
	if calls := crawledPages.indexablePages(); len(calls) != 0 {
		t.Fatalf("a not-modified fetch should publish no crawled page, got %v", calls)
	}
}

type silentMediaTypeObserver struct{}

func (silentMediaTypeObserver) MediaTypeUnparsed(context.Context, string, error) {}

type silentLinkResolutionObserver struct{}

func (silentLinkResolutionObserver) BaseURLUnresolved(
	context.Context,
	canonicalurl.CanonicalURL,
	string,
	error,
) {
}

func (silentLinkResolutionObserver) LinksUnresolved(
	context.Context,
	canonicalurl.CanonicalURL,
	int,
) {
}
