package pagevisit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkdiscovery"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

const (
	orderID              = "o1"
	pageLinkingNext      = `<html><body><a href="/next">next</a></body></html>`
	pageRefusingIndexing = `<html><head><meta name="robots" content="noindex">` +
		`</head><body><a href="/next">next</a></body></html>`
)

type fakeFetch struct {
	outcome    pagefetch.FetchOutcome
	gotVersion pagefetch.PageVersion
	sawFetch   bool
}

func (f *fakeFetch) Fetch(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	knownVersion pagefetch.PageVersion,
) pagefetch.FetchOutcome {
	f.gotVersion = knownVersion
	f.sawFetch = true
	return f.outcome
}

type visitedCall struct {
	url     canonicalurl.CanonicalURL
	version pagefetch.PageVersion
}

type fakePageVisitRecords struct {
	due          bool
	version      pagefetch.PageVersion
	visitedCalls []visitedCall
}

func (f *fakePageVisitRecords) LastPageVisitOf(
	context.Context,
	canonicalurl.CanonicalURL,
) (pagevisit.LastPageVisit, bool) {
	return pagevisit.LastPageVisit{Version: f.version}, true
}

func (f *fakePageVisitRecords) PageDueForRecrawl(pagevisit.LastPageVisit) bool {
	return f.due
}

func (f *fakePageVisitRecords) RecordPageVisit(
	_ context.Context,
	url canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) {
	f.visitedCalls = append(f.visitedCalls, visitedCall{url: url, version: version})
}

type fakeCrawledPages struct {
	indexable   []string
	refused     []string
	unreachable bool
}

func (f *fakeCrawledPages) PublishIndexablePage(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
) bool {
	f.indexable = append(f.indexable, pageURL.String())
	return !f.unreachable
}

func (f *fakeCrawledPages) PublishIndexingRefusedPage(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
) bool {
	f.refused = append(f.refused, pageURL.String())
	return !f.unreachable
}

type fakeFrontier struct {
	published []pendingpagevisit.PendingPageVisit
	err       error
}

func (f *fakeFrontier) Publish(
	_ context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
) error {
	if f.err != nil {
		return f.err
	}
	f.published = append(f.published, pageVisit)
	return nil
}

type unreadableHTMLPage struct{ err error }

func (page unreadableHTMLPage) ReadingOfPage(
	context.Context,
	canonicalurl.CanonicalURL,
	pagefetch.FetchedPage,
) (pagehtmlreading.Reading, error) {
	return pagehtmlreading.Reading{}, page.err
}

type pageVisitScene struct {
	fetch            *fakeFetch
	pageVisitRecords *fakePageVisitRecords
	htmlPageReading  pagevisit.HTMLPageReading
	crawledPages     *fakeCrawledPages
	frontier         *fakeFrontier
	profile          yacycrawlcontract.CrawlProfile
}

func sceneFetching(outcome pagefetch.FetchOutcome) *pageVisitScene {
	return &pageVisitScene{
		fetch:            &fakeFetch{outcome: outcome},
		pageVisitRecords: &fakePageVisitRecords{due: true},
		htmlPageReading: pagehtmlreading.NewHTMLPageReading(
			pagehtml.NewHTMLParser(silentMediaTypeObserver{}),
			linkdiscovery.NewLinkDiscovery(silentLinkResolutionObserver{}),
			silentRefusalEnforcementObserver{},
		),
		crawledPages: &fakeCrawledPages{},
		frontier:     &fakeFrontier{},
		profile: yacycrawlcontract.CrawlProfile{
			Scope:           yacycrawlcontract.ScopeWide,
			URLMustMatch:    yacycrawlcontract.MatchAll,
			MaxDepth:        5,
			MaxPagesPerHost: yacycrawlcontract.UnlimitedPagesPerHost,
		},
	}
}

func (scene *pageVisitScene) visit(t *testing.T) pagevisit.PageVisitOutcome {
	t.Helper()
	order, err := acceptedorder.AcceptedOrderFrom(yacycrawlcontract.CrawlOrder{
		OrderID: orderID, Profile: scene.profile,
	})
	if err != nil {
		t.Fatalf("order: %v", err)
	}
	return pagevisit.New(
		scene.fetch,
		scene.pageVisitRecords,
		scene.pageVisitRecords,
		scene.htmlPageReading,
		scene.crawledPages,
		scene.frontier,
	).VisitPage(t.Context(), order, pendingpagevisit.PendingPageVisit{
		OrderID: orderID,
		URL:     canonicalurltest.CanonicalURLOf(t, "http://host/"),
		Depth:   1,
	})
}

func (scene *pageVisitScene) completedVisit(t *testing.T) pagevisit.PageVisitOutcome {
	t.Helper()
	outcome := scene.visit(t)
	if outcome.Conclusion != pagevisit.PageVisitTerminal {
		t.Fatalf("conclusion = %v, want terminal", outcome.Conclusion)
	}
	return outcome
}

func fetchedOutcome(markup string) pagefetch.FetchOutcome {
	return pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			ContentType: "text/html",
			Body:        []byte(markup),
		},
		Version: pagefetch.PageVersion{EntityTag: `"etag"`},
	}
}

func TestAReadPageIsPublishedAndItsLinksGoOnTheFrontier(t *testing.T) {
	scene := sceneFetching(fetchedOutcome(pageLinkingNext))

	outcome := scene.completedVisit(t)

	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("a published page carries no disposal, got %q", outcome.Disposal)
	}
	if len(scene.crawledPages.indexable) != 1 || scene.crawledPages.indexable[0] != "http://host/" {
		t.Fatalf("want the page published as indexable once, got %v", scene.crawledPages.indexable)
	}
	want := pendingpagevisit.PendingPageVisit{
		OrderID: orderID,
		URL:     canonicalurltest.CanonicalURLOf(t, "http://host/next"),
		Depth:   2,
	}
	if len(scene.frontier.published) != 1 || scene.frontier.published[0] != want {
		t.Fatalf("frontier got %+v, want the link one level deeper", scene.frontier.published)
	}
}

func TestAPageThatRefusesIndexingIsPublishedAsRefusingIndexing(t *testing.T) {
	scene := sceneFetching(fetchedOutcome(pageRefusingIndexing))

	scene.completedVisit(t)

	if len(scene.crawledPages.refused) != 1 {
		t.Fatalf("want the page published as refusing indexing, got %v", scene.crawledPages.refused)
	}
	if len(scene.crawledPages.indexable) != 0 {
		t.Fatalf(
			"a page that refuses indexing is not indexable, got %v",
			scene.crawledPages.indexable,
		)
	}
}

func TestLinksBeyondTheProfileStayOffTheFrontier(t *testing.T) {
	scene := sceneFetching(fetchedOutcome(pageLinkingNext))
	scene.profile.MaxDepth = 1

	scene.completedVisit(t)

	if len(scene.frontier.published) != 0 {
		t.Fatalf("frontier got %v, want no link past the profile depth", scene.frontier.published)
	}
}

func TestARedirectTargetGoesOnTheFrontierAtTheSameDepth(t *testing.T) {
	redirectTarget := canonicalurltest.CanonicalURLOf(t, "http://host/moved")
	scene := sceneFetching(pagefetch.FetchOutcome{
		Status:         pagefetch.FetchRedirected,
		RedirectTarget: redirectTarget,
	})

	outcome := scene.completedVisit(t)

	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("a redirect carries no disposal reason, got %q", outcome.Disposal)
	}
	want := pendingpagevisit.PendingPageVisit{OrderID: orderID, URL: redirectTarget, Depth: 1}
	if len(scene.frontier.published) != 1 || scene.frontier.published[0] != want {
		t.Fatalf("frontier got %+v, want the redirect target", scene.frontier.published)
	}
	if len(scene.crawledPages.indexable) != 0 {
		t.Fatalf("a redirect publishes no crawled page, got %v", scene.crawledPages.indexable)
	}
	if len(scene.pageVisitRecords.visitedCalls) != 1 {
		t.Fatalf("recorded visits %v, want one", scene.pageVisitRecords.visitedCalls)
	}
}

func TestARedirectTargetBeyondTheProfileStaysOffTheFrontier(t *testing.T) {
	scene := sceneFetching(pagefetch.FetchOutcome{
		Status:         pagefetch.FetchRedirected,
		RedirectTarget: canonicalurltest.CanonicalURLOf(t, "http://host/moved"),
	})
	scene.profile.URLMustNotMatch = "/moved"

	scene.completedVisit(t)

	if len(scene.frontier.published) != 0 {
		t.Fatalf(
			"frontier got %v, want no redirect target past the profile",
			scene.frontier.published,
		)
	}
}

func TestAPageThatWasNotPublishedReturnsUnrecorded(t *testing.T) {
	scene := sceneFetching(fetchedOutcome(pageLinkingNext))
	scene.crawledPages.unreachable = true

	outcome := scene.visit(t)

	if outcome.Conclusion != pagevisit.PageVisitReturned || outcome.ReturnCause == nil {
		t.Fatalf("outcome = %+v, want returned with a cause", outcome)
	}
	if len(scene.pageVisitRecords.visitedCalls) != 0 {
		t.Fatalf("recorded visits %v, want none", scene.pageVisitRecords.visitedCalls)
	}
	if len(scene.frontier.published) != 0 {
		t.Fatalf(
			"frontier got %v, want nothing before the page is published",
			scene.frontier.published,
		)
	}
}

func TestLinksTheFrontierRefusesReturnTheVisitUnrecorded(t *testing.T) {
	frontierDown := errors.New("stream down")
	scene := sceneFetching(fetchedOutcome(pageLinkingNext))
	scene.frontier.err = frontierDown

	outcome := scene.visit(t)

	if outcome.Conclusion != pagevisit.PageVisitReturned ||
		!errors.Is(outcome.ReturnCause, frontierDown) {
		t.Fatalf("outcome = %+v, want returned for the frontier", outcome)
	}
	if len(scene.pageVisitRecords.visitedCalls) != 0 {
		t.Fatalf("recorded visits %v, want none", scene.pageVisitRecords.visitedCalls)
	}
}

func TestAReadPageRecordsTheVersionTheOriginAnswered(t *testing.T) {
	scene := sceneFetching(fetchedOutcome(pageLinkingNext))

	scene.completedVisit(t)

	calls := scene.pageVisitRecords.visitedCalls
	if len(calls) != 1 || calls[0].url.String() != "http://host/" ||
		calls[0].version.EntityTag != `"etag"` {
		t.Fatalf("recorded visits %+v, want the page at its fetched version", calls)
	}
}

func TestAPageThatIsNotHTMLIsDisposedAndRecorded(t *testing.T) {
	fetchOutcome := fetchedOutcome(pageLinkingNext)
	fetchOutcome.Page.ContentType = "application/pdf"
	scene := sceneFetching(fetchOutcome)

	outcome := scene.completedVisit(t)

	if outcome.Disposal != disposal.UnsupportedMediaType {
		t.Fatalf("disposal = %q, want unsupported media type", outcome.Disposal)
	}
	if len(scene.pageVisitRecords.visitedCalls) != 1 {
		t.Fatalf("recorded visits %v, want one", scene.pageVisitRecords.visitedCalls)
	}
	if len(scene.crawledPages.indexable) != 0 {
		t.Fatalf("a disposed page publishes nothing, got %v", scene.crawledPages.indexable)
	}
}

func TestUnreadablePageHTMLIsDisposed(t *testing.T) {
	scene := sceneFetching(fetchedOutcome(pageLinkingNext))
	scene.htmlPageReading = unreadableHTMLPage{err: errors.New("boom")}

	outcome := scene.completedVisit(t)

	if outcome.Disposal != disposal.UnreadableHTML {
		t.Fatalf("disposal = %q, want unreadable html", outcome.Disposal)
	}
}

func TestAFetchThatEndsWithoutAPageConcludesByItsStatus(t *testing.T) {
	for status, want := range map[pagefetch.FetchStatus]disposal.Reason{
		pagefetch.FetchRejected:              disposal.FetchRejected,
		pagefetch.FetchGone:                  disposal.FetchRejected,
		pagefetch.FetchOversized:             disposal.Oversized,
		pagefetch.FetchRedirectTargetInvalid: disposal.RedirectTargetInvalid,
		pagefetch.FetchAccessRefused:         disposal.AccessRefused,
		pagefetch.FetchNotModified:           disposal.NotModified,
	} {
		scene := sceneFetching(pagefetch.FetchOutcome{Status: status})

		outcome := scene.completedVisit(t)

		if outcome.Disposal != want {
			t.Fatalf("status %v: disposal = %q, want %q", status, outcome.Disposal, want)
		}
		if len(scene.crawledPages.indexable) != 0 || len(scene.frontier.published) != 0 {
			t.Fatalf("status %v: want nothing published", status)
		}
	}
}

func TestAVisitIsRecordedOnlyWhenItStartsTheRecrawlGrace(t *testing.T) {
	for status, wantRecorded := range map[pagefetch.FetchStatus]bool{
		pagefetch.FetchRejected:              false,
		pagefetch.FetchOversized:             false,
		pagefetch.FetchRedirectTargetInvalid: false,
		pagefetch.FetchFailed:                false,
		pagefetch.FetchDeferred:              false,
		pagefetch.FetchAccessRefused:         true,
		pagefetch.FetchNotModified:           true,
	} {
		scene := sceneFetching(pagefetch.FetchOutcome{Status: status})

		scene.visit(t)

		if recorded := len(scene.pageVisitRecords.visitedCalls) == 1; recorded != wantRecorded {
			t.Fatalf("status %v: recorded = %v, want %v", status, recorded, wantRecorded)
		}
	}
}

func TestAFailedFetchLeavesThePageForAnotherAttempt(t *testing.T) {
	for _, status := range []pagefetch.FetchStatus{
		pagefetch.FetchFailed, pagefetch.FetchDeadlinePassed, pagefetch.FetchStatus(99),
	} {
		outcome := sceneFetching(pagefetch.FetchOutcome{Status: status}).visit(t)

		if outcome.Conclusion != pagevisit.PageVisitRetryable {
			t.Fatalf("status %v: conclusion = %v, want retryable", status, outcome.Conclusion)
		}
	}
}

func TestADeferredFetchDefersTheVisit(t *testing.T) {
	outcome := sceneFetching(pagefetch.FetchOutcome{
		Status: pagefetch.FetchDeferred, DeferFor: time.Second,
	}).visit(t)

	if outcome.Conclusion != pagevisit.PageVisitDeferred || outcome.DeferFor != time.Second {
		t.Fatalf("want deferred for 1s, got %+v", outcome)
	}
}

func TestAPageNotDueIsNotFetched(t *testing.T) {
	scene := sceneFetching(fetchedOutcome(pageLinkingNext))
	scene.pageVisitRecords.due = false

	outcome := scene.completedVisit(t)

	if outcome.Disposal != disposal.NotDue {
		t.Fatalf("disposal = %q, want not due", outcome.Disposal)
	}
	if scene.fetch.sawFetch {
		t.Fatal("a page not due was fetched")
	}
}

func TestTheFetchAsksForTheVersionLastVisited(t *testing.T) {
	known := pagefetch.PageVersion{EntityTag: `"stored-etag"`}
	scene := sceneFetching(fetchedOutcome(pageLinkingNext))
	scene.pageVisitRecords.version = known

	scene.visit(t)

	if scene.fetch.gotVersion != known {
		t.Fatalf("fetched version = %+v, want %+v", scene.fetch.gotVersion, known)
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

type silentRefusalEnforcementObserver struct{}

func (silentRefusalEnforcementObserver) LinkDiscoveryRefusalEnforced(
	context.Context,
	canonicalurl.CanonicalURL,
) {
}
