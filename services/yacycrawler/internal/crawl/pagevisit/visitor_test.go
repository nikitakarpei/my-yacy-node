package pagevisit_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/scraperequestpublication"
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
	_ string,
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

type fakeExtract struct {
	document contentextraction.ExtractedDocument
	err      error
}

func (f fakeExtract) DocumentFrom(
	_ context.Context,
	_, _ string,
	_ []byte,
) (contentextraction.ExtractedDocument, error) {
	return f.document, f.err
}

func readableDocument() contentextraction.ExtractedDocument {
	return contentextraction.ExtractedDocument{
		Title:  "title",
		Body:   []byte("body"),
		Format: contentformatgraph.FormatReadableText,
	}
}

func refusingDocument() contentextraction.ExtractedDocument {
	return contentextraction.ExtractedDocument{
		Body:            []byte("b"),
		Format:          contentformatgraph.FormatReadableText,
		RefusesIndexing: true,
	}
}

func linkingDocument(t *testing.T, discovered string) contentextraction.ExtractedDocument {
	t.Helper()
	return contentextraction.ExtractedDocument{
		Body:   []byte("b"),
		Format: contentformatgraph.FormatReadableText,
		DiscoveredURLs: []canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, discovered),
		},
	}
}

func readableExtract() fakeExtract {
	return fakeExtract{document: readableDocument()}
}

type recordingObserver struct {
	mu                      sync.Mutex
	refusals                map[refusal.Demand]int
	fetched                 int
	scrapeRequestsPublished int
}

func newObserver() *recordingObserver {
	return &recordingObserver{refusals: map[refusal.Demand]int{}}
}

func (o *recordingObserver) PageFetched() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.fetched++
}

func (o *recordingObserver) RefusalHonored(kind refusal.Demand) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.refusals[kind]++
}

func (o *recordingObserver) ScrapeRequestPublished() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.scrapeRequestsPublished++
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

func fetchedOutcome() pagefetch.FetchOutcome {
	return pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			FinalURL:    "http://host/",
			ContentType: "text/html",
			Body:        []byte("x"),
		},
		Version: pagefetch.PageVersion{EntityTag: `"etag"`},
	}
}

func fetchOf(outcome pagefetch.FetchOutcome) *fakeFetch {
	return &fakeFetch{outcome: outcome}
}

func newVisitor(
	fetcher pagefetch.Fetcher,
	recrawl pagevisit.RecrawlRule,
	extractor pagevisit.PageExtractor,
	observer *recordingObserver,
	scrapeRequests scraperequestpublication.ScrapeRequests,
) pagevisit.Visitor {
	return newVisitorSource(fetcher, recrawl, extractor, observer, scrapeRequests).
		VisitorFor(pagevisit.Honored)
}

func newVisitorSource(
	fetcher pagefetch.Fetcher,
	recrawl pagevisit.RecrawlRule,
	extractor pagevisit.PageExtractor,
	observer *recordingObserver,
	scrapeRequests scraperequestpublication.ScrapeRequests,
) pagevisit.VisitorSource {
	return pagevisit.New(
		fetcher,
		recrawl,
		extractor,
		observer,
		scraperequestpublication.NewPublisher(observer, scrapeRequests),
	)
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

func TestVisitAbsorbsFetchedPage(t *testing.T) {
	observer := newObserver()
	scrapeRequests := &fakeScrapeRequests{}
	visitor := newVisitor(
		fetchOf(fetchedOutcome()),
		&fakeRecrawl{due: true},
		fakeExtract{document: linkingDocument(t, "http://host/next")},
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
	if !outcome.Fetched {
		t.Fatal("an absorbed page counts as fetched")
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

func TestVisitReportsNotAPageDisposal(t *testing.T) {
	recrawl := &fakeRecrawl{due: true}
	scrapeRequests := &fakeScrapeRequests{}
	visitor := newVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchNotAPage}),
		recrawl,
		readableExtract(),
		newObserver(),
		scrapeRequests,
	)

	outcome := visitHost(t, visitor)

	if outcome.Disposal != disposal.NotAPage {
		t.Fatalf("want not-a-page disposal, got %q", outcome.Disposal)
	}
	if !outcome.Fetched {
		t.Fatal("a fetched body that is not a page still consumes the budget")
	}
	if len(recrawl.calls()) != 0 {
		t.Fatalf("visited should not be recorded for not-a-page, got %v", recrawl.calls())
	}
	if calls := scrapeRequests.calls(); len(calls) != 0 {
		t.Fatalf("a not-a-page fetch should publish no scrape request, got %v", calls)
	}
}

func TestVisitCeasesOnHTTPCease(t *testing.T) {
	observer := newObserver()
	recrawl := &fakeRecrawl{due: true}
	scrapeRequests := &fakeScrapeRequests{}
	visitor := newVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchCeased}),
		recrawl,
		readableExtract(),
		observer,
		scrapeRequests,
	)

	outcome := visitHost(t, visitor)

	if observer.refusals[refusal.Cease] != 1 {
		t.Fatalf("cease not honored: %v", observer.refusals)
	}
	if outcome.Disposal != disposal.Refused {
		t.Fatalf("want refused disposal, got %q", outcome.Disposal)
	}
	if outcome.Fetched {
		t.Fatal("a refused fetch returns no body and must not consume the budget")
	}
	if len(recrawl.calls()) != 1 {
		t.Fatalf("visited should be recorded on refusal so grace applies, got %v", recrawl.calls())
	}
	if calls := scrapeRequests.calls(); len(calls) != 0 {
		t.Fatalf("a ceased fetch should publish no scrape request, got %v", calls)
	}
}

func TestVisitReportsTransient(t *testing.T) {
	recrawl := &fakeRecrawl{due: true}
	visitor := newVisitor(
		fetchOf(pagefetch.FetchOutcome{Status: pagefetch.FetchFailed}),
		recrawl,
		readableExtract(),
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
		readableExtract(),
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
		readableExtract(),
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
		readableExtract(),
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

func TestVisitRecrawlDecisionErrorFails(t *testing.T) {
	visitor := newVisitor(
		&fakeFetch{},
		&fakeRecrawl{err: errors.New("boom")},
		readableExtract(),
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
	fetch := fetchOf(fetchedOutcome())
	visitor := newVisitor(
		fetch,
		&fakeRecrawl{due: false},
		readableExtract(),
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
	if outcome.Fetched {
		t.Fatal("a skipped fetch must not consume the budget")
	}
}

func TestVisitPassesKnownVersionToFetcher(t *testing.T) {
	fetch := fetchOf(fetchedOutcome())
	known := pagefetch.PageVersion{EntityTag: `"stored-etag"`}
	visitor := newVisitor(
		fetch,
		&fakeRecrawl{due: true, version: known},
		readableExtract(),
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

func TestVisitRecordsVersionAfterAbsorbingThePage(t *testing.T) {
	recrawl := &fakeRecrawl{due: true}
	visitor := newVisitor(
		fetchOf(fetchedOutcome()),
		recrawl,
		readableExtract(),
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
		fetchOf(fetchedOutcome()),
		recrawl,
		fakeExtract{err: errors.New("parser broke")},
		newObserver(),
		scrapeRequests,
	)

	outcome := visitHost(t, visitor)

	if outcome.Disposal != disposal.Unextractable {
		t.Fatalf("want the absorbed reason reported, got %q", outcome.Disposal)
	}
	if len(recrawl.calls()) != 1 {
		t.Fatalf("want the visit recorded regardless of publication, got %v", recrawl.calls())
	}
	if calls := scrapeRequests.calls(); len(calls) != 0 {
		t.Fatalf("a disposed page should publish no scrape request, got %v", calls)
	}
}

func TestVisitNotModifiedRecordsVersionWithoutAbsorbing(t *testing.T) {
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
		readableExtract(),
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
		fetchOf(fetchedOutcome()),
		recrawl,
		readableExtract(),
		newObserver(),
		&fakeScrapeRequests{},
	)

	visitHost(t, visitor)
}

func TestVisitScrapeRequestPublishErrorFails(t *testing.T) {
	scrapeRequests := &fakeScrapeRequests{err: errors.New("publish boom")}
	visitor := newVisitor(
		fetchOf(fetchedOutcome()),
		&fakeRecrawl{due: true},
		readableExtract(),
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
