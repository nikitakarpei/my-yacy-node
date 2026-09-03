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
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagerefusals"
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

type fakeRecrawl struct {
	mu      sync.Mutex
	due     bool
	version pagefetch.PageVersion
	err     error

	visitedErr   error
	visitedCalls []visitedCall
}

func (f *fakeRecrawl) DecisionFor(
	context.Context,
	canonicalurl.CanonicalURL,
) (pagevisit.RecrawlDecision, error) {
	if f.err != nil {
		return pagevisit.RecrawlDecision{}, f.err
	}
	return pagevisit.RecrawlDecision{Due: f.due, Version: f.version}, nil
}

func (f *fakeRecrawl) RecordVisit(
	_ context.Context,
	url canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.visitedCalls = append(
		f.visitedCalls,
		visitedCall{url: url, version: version},
	)
	return f.visitedErr
}

func (f *fakeRecrawl) calls() []visitedCall {
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
	mu                      sync.Mutex
	refusals                map[string]int
	fetchDurations          []time.Duration
	fetched                 int
	scrapeRequestsPublished int
}

func newObserver() *recordingObserver {
	return &recordingObserver{refusals: map[string]int{}}
}

func (o *recordingObserver) PageFetchCompleted(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ pagefetch.FetchStatus,
	duration time.Duration,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.fetchDurations = append(o.fetchDurations, duration)
	o.fetched++
}

func (*recordingObserver) FetchConcluded(
	context.Context,
	canonicalurl.CanonicalURL,
	pagefetch.FetchStatus,
) {
}

func (o *recordingObserver) PageFetchFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
	_ error,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.fetchDurations = append(o.fetchDurations, duration)
}

func (o *recordingObserver) IndexingRefusalEnforced(
	context.Context,
	canonicalurl.CanonicalURL,
) {
	o.honor("indexing")
}

func (o *recordingObserver) LinkDiscoveryRefusalEnforced(
	context.Context,
	canonicalurl.CanonicalURL,
) {
	o.honor("link-discovery")
}

func (o *recordingObserver) honor(refusal string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.refusals[refusal]++
}

func (o *recordingObserver) ScrapeRequestPublished(
	context.Context,
	canonicalurl.CanonicalURL,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.scrapeRequestsPublished++
}

func (*recordingObserver) ScrapeRequestPublicationFailed(
	context.Context,
	canonicalurl.CanonicalURL,
	error,
) {
}

func (*recordingObserver) RecrawlRecordFailed(
	context.Context,
	canonicalurl.CanonicalURL,
	error,
) {
}

func (*recordingObserver) PageHTMLUnreadable(
	context.Context,
	canonicalurl.CanonicalURL,
	error,
) {
}

type fakeScrapeRequests struct {
	mu        sync.Mutex
	err       error
	published []string
}

func (f *fakeScrapeRequests) Publish(
	_ context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.published = append(f.published, canonicalURL.String())
	return nil
}

func (f *fakeScrapeRequests) calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.published...)
}

func fetchedOutcome(t *testing.T) pagefetch.FetchOutcome {
	t.Helper()
	return pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			LandedURL:   canonicalurltest.CanonicalURLOf(t, "http://host/"),
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

func newVisitor(
	fetcher pagefetch.Fetcher,
	recrawl pagevisit.RecrawlRule,
	observer *recordingObserver,
	scrapeRequests pagevisit.ScrapeRequests,
) pagevisit.Visitor {
	return newVisitorFor(fetcher, recrawl, observer, scrapeRequests)(
		pagerefusals.IgnoredRefusals{})
}

func newVisitorFor(
	fetcher pagefetch.Fetcher,
	recrawl pagevisit.RecrawlRule,
	observer *recordingObserver,
	scrapeRequests pagevisit.ScrapeRequests,
) pagevisit.VisitorFor {
	pageFetcher := pagevisit.NewPageFetcher(
		fetcher, &steppingClock{now: time.Unix(0, 0), step: fetchStep}, observer,
	)
	recrawlRule := pagevisit.NewBestEffortRecrawlRule(recrawl, observer)
	scrapePublisher := pagevisit.NewScrapeRequestPublisher(scrapeRequests, observer)
	return pagevisit.New(pageFetcher, recrawlRule, observer, scrapePublisher)
}

func visitHost(t *testing.T, visitor pagevisit.Visitor) pagevisit.VisitOutcome {
	t.Helper()
	outcome, err := visitor.Visit(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Conclusion != pagevisit.VisitCompleted {
		t.Fatalf("want concluded, got %v", outcome.Conclusion)
	}
	return outcome
}

func TestVisitReadsTheFetchedPage(t *testing.T) {
	observer := newObserver()
	scrapeRequests := &fakeScrapeRequests{}
	visitor := newVisitor(
		fetchOf(fetchedOutcome(t)),
		&fakeRecrawl{due: true},
		observer,
		scrapeRequests,
	)

	outcome := visitHost(t, visitor)

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
	if calls := scrapeRequests.calls(); len(calls) != 1 || calls[0] != "http://host/" {
		t.Fatalf("want the scrape request published once with its canonical url, got %v", calls)
	}
	if observer.scrapeRequestsPublished != 1 {
		t.Fatalf(
			"want the scrape request metric observed once, got %d",
			observer.scrapeRequestsPublished,
		)
	}
}

func TestVisitReportsFetchRejectedDisposal(t *testing.T) {
	recrawl := &fakeRecrawl{due: true}
	scrapeRequests := &fakeScrapeRequests{}
	visitor := newVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchRejected}),
		recrawl,
		newObserver(),
		scrapeRequests,
	)

	outcome := visitHost(t, visitor)

	if outcome.Disposal != disposal.FetchRejected {
		t.Fatalf("want fetch-rejected disposal, got %q", outcome.Disposal)
	}
	if len(recrawl.calls()) != 0 {
		t.Fatalf("visited should not be recorded for a rejected fetch, got %v", recrawl.calls())
	}
	if calls := scrapeRequests.calls(); len(calls) != 0 {
		t.Fatalf("a rejected fetch should publish no scrape request, got %v", calls)
	}
}

func TestVisitReportsOversizedDisposal(t *testing.T) {
	scrapeRequests := &fakeScrapeRequests{}
	visitor := newVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchOversized}),
		&fakeRecrawl{due: true},
		newObserver(),
		scrapeRequests,
	)

	outcome := visitHost(t, visitor)

	if outcome.Disposal != disposal.Oversized {
		t.Fatalf("want oversized disposal, got %q", outcome.Disposal)
	}
	if calls := scrapeRequests.calls(); len(calls) != 0 {
		t.Fatalf("an oversized page should publish no scrape request, got %v", calls)
	}
}

func TestVisitReportsLandedURLInvalidDisposal(t *testing.T) {
	scrapeRequests := &fakeScrapeRequests{}
	visitor := newVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchLandedURLInvalid}),
		&fakeRecrawl{due: true},
		newObserver(),
		scrapeRequests,
	)

	outcome := visitHost(t, visitor)

	if outcome.Disposal != disposal.LandedURLInvalid {
		t.Fatalf("want landed-url-invalid disposal, got %q", outcome.Disposal)
	}
	if calls := scrapeRequests.calls(); len(calls) != 0 {
		t.Fatalf("an invalid landing should publish no scrape request, got %v", calls)
	}
}

func TestVisitStopsWhenTheTargetRefusesAccess(t *testing.T) {
	observer := newObserver()
	recrawl := &fakeRecrawl{due: true}
	scrapeRequests := &fakeScrapeRequests{}
	visitor := newVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchAccessRefused}),
		recrawl,
		observer,
		scrapeRequests,
	)

	outcome := visitHost(t, visitor)

	if outcome.Disposal != disposal.AccessRefused {
		t.Fatalf("want access-refused disposal, got %q", outcome.Disposal)
	}
	if len(recrawl.calls()) != 1 {
		t.Fatalf(
			"visited should be recorded on a refusal so grace applies, got %v",
			recrawl.calls(),
		)
	}
	if calls := scrapeRequests.calls(); len(calls) != 0 {
		t.Fatalf("a refused fetch should publish no scrape request, got %v", calls)
	}
}

func TestVisitReportsTransient(t *testing.T) {
	recrawl := &fakeRecrawl{due: true}
	visitor := newVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchFailed}),
		recrawl,
		newObserver(),
		&fakeScrapeRequests{},
	)

	outcome, err := visitor.Visit(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Conclusion != pagevisit.VisitRetryable {
		t.Fatalf("want retryable, got %v", outcome.Conclusion)
	}
	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("a transient failure must not dispose, got %q", outcome.Disposal)
	}
	if len(recrawl.calls()) != 0 {
		t.Fatalf("visited should not be recorded after failure, got %v", recrawl.calls())
	}
}

func TestVisitUnknownFetchStatusFails(t *testing.T) {
	visitor := newVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchStatus(99)}),
		&fakeRecrawl{due: true},
		newObserver(),
		&fakeScrapeRequests{},
	)

	if _, err := visitor.Visit(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	); err == nil {
		t.Fatal("an unknown fetch status should fail the visit, not retry silently")
	}
}

func TestVisitReportsDeferred(t *testing.T) {
	visitor := newVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchDeferred, DeferFor: time.Second}),
		&fakeRecrawl{due: true},
		newObserver(),
		&fakeScrapeRequests{},
	)

	outcome, err := visitor.Visit(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	if outcome.Conclusion != pagevisit.VisitDeferred || outcome.DeferFor != time.Second {
		t.Fatalf("want deferred for 1s, got %+v", outcome)
	}
}

func TestVisitFetchErrorFails(t *testing.T) {
	visitor := newVisitor(
		&fakeFetch{err: errors.New("boom")},
		&fakeRecrawl{due: true},
		newObserver(),
		&fakeScrapeRequests{},
	)

	if _, err := visitor.Visit(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	); err == nil {
		t.Fatal("fetch error should fail the visit")
	}
}

func TestVisitReportsHowLongTheFetchTook(t *testing.T) {
	observer := newObserver()

	visitHost(t, newVisitor(
		fetchOf(fetchedOutcome(t)),
		&fakeRecrawl{due: true},
		observer,
		&fakeScrapeRequests{},
	))

	if len(observer.fetchDurations) != 1 || observer.fetchDurations[0] != fetchStep {
		t.Fatalf("want one %v fetch observed, got %v", fetchStep, observer.fetchDurations)
	}
}

func TestVisitReportsHowLongAFailedFetchTook(t *testing.T) {
	observer := newObserver()
	visitor := newVisitor(
		&fakeFetch{err: errors.New("boom")},
		&fakeRecrawl{due: true},
		observer,
		&fakeScrapeRequests{},
	)

	_, _ = visitor.Visit(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)

	if len(observer.fetchDurations) != 1 || observer.fetchDurations[0] != fetchStep {
		t.Fatalf("want one %v fetch observed, got %v", fetchStep, observer.fetchDurations)
	}
}

func TestVisitReportsNoFetchDurationWhenNotDue(t *testing.T) {
	observer := newObserver()

	visitHost(t, newVisitor(
		fetchOf(fetchedOutcome(t)),
		&fakeRecrawl{due: false},
		observer,
		&fakeScrapeRequests{},
	))

	if len(observer.fetchDurations) != 0 {
		t.Fatalf("want no fetch observed, got %v", observer.fetchDurations)
	}
}

func TestVisitRecrawlDecisionErrorFails(t *testing.T) {
	visitor := newVisitor(
		&fakeFetch{},
		&fakeRecrawl{err: errors.New("boom")},
		newObserver(),
		&fakeScrapeRequests{},
	)

	if _, err := visitor.Visit(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	); err == nil {
		t.Fatal("recrawl decision error should fail the visit")
	}
}

func TestVisitReportsNotDueWithoutFetching(t *testing.T) {
	fetch := fetchOf(fetchedOutcome(t))
	visitor := newVisitor(
		fetch,
		&fakeRecrawl{due: false},
		newObserver(),
		&fakeScrapeRequests{},
	)

	outcome := visitHost(t, visitor)

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
	visitor := newVisitor(
		fetch,
		&fakeRecrawl{due: true, version: known},
		newObserver(),
		&fakeScrapeRequests{},
	)

	if _, err := visitor.Visit(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	); err != nil {
		t.Fatalf("visit: %v", err)
	}
	if fetch.gotVersion != known {
		t.Fatalf("fetcher version = %+v, want %+v", fetch.gotVersion, known)
	}
}

func TestVisitRecordsVersionAfterReadingThePage(t *testing.T) {
	recrawl := &fakeRecrawl{due: true}
	visitor := newVisitor(
		fetchOf(fetchedOutcome(t)),
		recrawl,
		newObserver(),
		&fakeScrapeRequests{},
	)

	if _, err := visitor.Visit(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	); err != nil {
		t.Fatalf("visit: %v", err)
	}
	calls := recrawl.calls()
	if len(calls) != 1 {
		t.Fatalf("want exactly one visited call, got %v", calls)
	}
	if calls[0].url.String() != "http://host/" || calls[0].version.EntityTag != `"etag"` {
		t.Fatalf("visited call = %+v", calls[0])
	}
}

func TestVisitReportsTheDisposalAbsorptionReached(t *testing.T) {
	recrawl := &fakeRecrawl{due: true}
	scrapeRequests := &fakeScrapeRequests{}
	visitor := newVisitor(
		fetchOf(unreadableOutcome(t)),
		recrawl,
		newObserver(),
		scrapeRequests,
	)

	outcome := visitHost(t, visitor)

	if outcome.Disposal != disposal.UnsupportedMediaType {
		t.Fatalf("want the page content reason reported, got %q", outcome.Disposal)
	}
	if len(recrawl.calls()) != 1 {
		t.Fatalf("want the visit recorded regardless of publication, got %v", recrawl.calls())
	}
	if calls := scrapeRequests.calls(); len(calls) != 0 {
		t.Fatalf("a disposed page should publish no scrape request, got %v", calls)
	}
}

func TestVisitNotModifiedRecordsVersionWithoutReadingThePage(t *testing.T) {
	recrawl := &fakeRecrawl{due: true}
	scrapeRequests := &fakeScrapeRequests{}
	visitor := newVisitor(
		fetchOf(
			pagefetch.FetchOutcome{
				Status:  pagefetch.FetchNotModified,
				Version: pagefetch.PageVersion{EntityTag: `"same"`},
			},
		),
		recrawl,
		newObserver(),
		scrapeRequests,
	)

	outcome := visitHost(t, visitor)

	if outcome.Disposal != disposal.NotModified {
		t.Fatalf("want not-modified disposal, got %q", outcome.Disposal)
	}
	calls := recrawl.calls()
	if len(calls) != 1 || calls[0].version.EntityTag != `"same"` {
		t.Fatalf("want the version recorded once, got %v", calls)
	}
	if calls := scrapeRequests.calls(); len(calls) != 0 {
		t.Fatalf("a not-modified fetch should publish no scrape request, got %v", calls)
	}
}

func TestVisitedErrorIsRecoverable(t *testing.T) {
	recrawl := &fakeRecrawl{due: true, visitedErr: errors.New("bucket down")}
	visitor := newVisitor(
		fetchOf(fetchedOutcome(t)),
		recrawl,
		newObserver(),
		&fakeScrapeRequests{},
	)

	visitHost(t, visitor)
}

func TestVisitScrapeRequestPublishErrorFails(t *testing.T) {
	scrapeRequests := &fakeScrapeRequests{err: errors.New("publish boom")}
	visitor := newVisitor(
		fetchOf(fetchedOutcome(t)),
		&fakeRecrawl{due: true},
		newObserver(),
		scrapeRequests,
	)

	if _, err := visitor.Visit(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	); err == nil {
		t.Fatal("a scrape request publish error should fail the visit")
	}
}
