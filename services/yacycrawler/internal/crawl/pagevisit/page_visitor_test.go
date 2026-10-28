package pagevisit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/yacycrawlcontracttest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkdiscovery"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagereading"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

const (
	orderID              = "o1"
	pageLinkingNext      = `<html><body><a href="/next">next</a></body></html>`
	pageLinkingTwoPages  = `<html><body><a href="/one">one</a><a href="/two">two</a></body></html>`
	pageRefusingIndexing = `<html><head><meta name="robots" content="noindex">` +
		`</head><body><a href="/next">next</a></body></html>`
)

var crawlID = crawlid.CrawlID{1}

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
	askedGrace   yacycrawlcontract.RecrawlGrace
	visitedCalls []visitedCall
	readErr      error
	recordErr    error
}

func (f *fakePageVisitRecords) LastPageVisitOf(
	*vault.Txn,
	canonicalurl.CanonicalURL,
) (pagevisit.LastPageVisit, bool, error) {
	if f.readErr != nil {
		return pagevisit.LastPageVisit{}, false, f.readErr
	}
	return pagevisit.LastPageVisit{Version: f.version}, true, nil
}

func (f *fakePageVisitRecords) RecrawlOf(
	_ pagevisit.LastPageVisit,
	recrawlGrace yacycrawlcontract.RecrawlGrace,
) pagevisit.Recrawl {
	f.askedGrace = recrawlGrace
	return pagevisit.Recrawl{Due: f.due, KnownVersion: f.version}
}

func (f *fakePageVisitRecords) RecordPageVisit(
	_ *vault.Txn,
	url canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) error {
	if f.recordErr != nil {
		return f.recordErr
	}
	f.visitedCalls = append(f.visitedCalls, visitedCall{url: url, version: version})
	return nil
}

type recordedRecordFailures struct {
	notRead     []error
	notRecorded []error
}

func (f *recordedRecordFailures) LastPageVisitNotRead(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	cause error,
) {
	f.notRead = append(f.notRead, cause)
}

func (f *recordedRecordFailures) PageVisitNotRecorded(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	cause error,
) {
	f.notRecorded = append(f.notRecorded, cause)
}

type fakeCrawledPages struct {
	indexable   []yacycrawlcontract.CrawledPage
	refused     []yacycrawlcontract.CrawledPage
	unreachable bool
}

func (f *fakeCrawledPages) PublishIndexablePage(
	_ context.Context,
	page yacycrawlcontract.CrawledPage,
) bool {
	f.indexable = append(f.indexable, page)
	return !f.unreachable
}

func (f *fakeCrawledPages) PublishIndexingRefusedPage(
	_ context.Context,
	page yacycrawlcontract.CrawledPage,
) bool {
	f.refused = append(f.refused, page)
	return !f.unreachable
}

type fakeFrontier struct {
	visitsLedTo  []pendingpagevisit.PendingPageVisit
	published    []pendingpagevisit.PendingPageVisit
	publications int
	err          error
}

func (f *fakeFrontier) AddNextOf(
	_ *vault.Txn,
	pageVisit pendingpagevisit.PendingPageVisit,
	pageVisits []pendingpagevisit.PendingPageVisit,
) error {
	if f.err != nil {
		return f.err
	}
	f.visitsLedTo = append(f.visitsLedTo, pageVisit)
	f.publications++
	f.published = append(f.published, pageVisits...)
	return nil
}

type unreadableHTMLPage struct{ err error }

func (page unreadableHTMLPage) ReadingOfPage(
	context.Context,
	canonicalurl.CanonicalURL,
	pagefetch.FetchedPage,
) (pagereading.Reading, error) {
	return pagereading.Reading{}, page.err
}

type countedUnadmittedLinks struct {
	amountOfLinks int
}

func (links *countedUnadmittedLinks) LinksUnadmitted(
	_ context.Context,
	_ crawlid.CrawlID,
	amountOfLinks int,
) {
	links.amountOfLinks += amountOfLinks
}

type pageVisitScene struct {
	fetch            *fakeFetch
	pageVisitRecords *fakePageVisitRecords
	recordFailures   *recordedRecordFailures
	pageReading      pagevisit.PageReading
	crawledPages     *fakeCrawledPages
	frontier         *fakeFrontier
	unadmittedLinks  *countedUnadmittedLinks
	profile          yacycrawlcontract.CrawlProfile
}

func sceneFetching(outcome pagefetch.FetchOutcome) *pageVisitScene {
	return &pageVisitScene{
		fetch:            &fakeFetch{outcome: outcome},
		pageVisitRecords: &fakePageVisitRecords{due: true},
		recordFailures:   &recordedRecordFailures{},
		pageReading: pagereading.NewPageReading(
			pagehtmlreading.NewHTMLPageReading(
				pagehtml.NewHTMLParser(silentMediaTypeObserver{}),
				linkdiscovery.NewLinkDiscovery(silentLinkResolutionObserver{}),
			),
			silentRefusalEnforcementObserver{},
		),
		crawledPages:    &fakeCrawledPages{},
		frontier:        &fakeFrontier{},
		unadmittedLinks: &countedUnadmittedLinks{},
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
		OrderID:  orderID,
		Profile:  scene.profile,
		SeedURLs: []canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://host/")},
	})
	if err != nil {
		t.Fatalf("order: %v", err)
	}
	storage, err := vault.New(memoryvault.OpenEngine(0), nil, nil, frozenClock{})
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	return pagevisit.New(
		storage,
		scene.fetch,
		scene.pageVisitRecords,
		scene.pageVisitRecords,
		scene.recordFailures,
		scene.pageReading,
		scene.crawledPages,
		scene.frontier,
		scene.unadmittedLinks,
	).VisitPage(t.Context(), order, pendingpagevisit.PendingPageVisit{
		CrawlID: crawlID,
		URL:     canonicalurltest.CanonicalURLOf(t, "http://host/"),
		Depth:   1,
	})
}

type frozenClock struct{}

func (frozenClock) Now() time.Time { return time.Unix(1_000_000, 0) }

func (frozenClock) Sleep(context.Context, time.Duration) error { return context.Canceled }

func (scene *pageVisitScene) completedVisit(t *testing.T) pagevisit.PageVisitOutcome {
	t.Helper()
	outcome := scene.visit(t)
	if outcome.Conclusion != pagevisit.PageVisitCompleted {
		t.Fatalf("conclusion = %v, want completed", outcome.Conclusion)
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

	if outcome.Result != yacycrawlcontract.ResultPublished {
		t.Fatalf("result = %q, want published", outcome.Result)
	}
	wantPage := yacycrawlcontract.CrawledPage{
		PageURL:     canonicalurltest.CanonicalURLOf(t, "http://host/"),
		ContentType: "text/html",
	}
	if len(scene.crawledPages.indexable) != 1 || scene.crawledPages.indexable[0] != wantPage {
		t.Fatalf("want the page published as indexable once, got %v", scene.crawledPages.indexable)
	}
	want := pendingpagevisit.PendingPageVisit{
		CrawlID: crawlID,
		URL:     canonicalurltest.CanonicalURLOf(t, "http://host/next"),
		Depth:   2,
	}
	if len(scene.frontier.published) != 1 || scene.frontier.published[0] != want {
		t.Fatalf("frontier got %+v, want the link one level deeper", scene.frontier.published)
	}
}

func TestTheLinksOfAPageGoOnTheFrontierInOnePublication(t *testing.T) {
	scene := sceneFetching(fetchedOutcome(pageLinkingTwoPages))

	scene.completedVisit(t)

	if len(scene.frontier.published) != 2 || scene.frontier.publications != 1 {
		t.Fatalf(
			"frontier got %d links in %d publications, want 2 in one",
			len(scene.frontier.published),
			scene.frontier.publications,
		)
	}
}

func TestALinkTheProfileDoesNotAdmitIsCountedAndStaysOffTheFrontier(t *testing.T) {
	scene := sceneFetching(fetchedOutcome(pageLinkingNext))
	scene.profile.MaxDepth = 1

	scene.completedVisit(t)

	if len(scene.frontier.published) != 0 {
		t.Fatalf(
			"frontier got %+v, want no link beyond the profile depth",
			scene.frontier.published,
		)
	}
	if scene.unadmittedLinks.amountOfLinks != 1 {
		t.Fatalf("unadmitted links = %d, want 1", scene.unadmittedLinks.amountOfLinks)
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

	if outcome.Result != yacycrawlcontract.ResultRedirected {
		t.Fatalf("result = %q, want redirected", outcome.Result)
	}
	want := pendingpagevisit.PendingPageVisit{CrawlID: crawlID, URL: redirectTarget, Depth: 1}
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

func fetchedOutcomeNotHTML() pagefetch.FetchOutcome {
	fetchOutcome := fetchedOutcome(pageLinkingNext)
	fetchOutcome.Page.ContentType = "application/pdf"
	return fetchOutcome
}

func TestAPageThatIsNotHTMLIsPublishedWithItsContentTypeAndRecorded(t *testing.T) {
	scene := sceneFetching(fetchedOutcomeNotHTML())

	outcome := scene.completedVisit(t)

	if outcome.Result != yacycrawlcontract.ResultPublished {
		t.Fatalf("result = %q, want published", outcome.Result)
	}
	wantPage := yacycrawlcontract.CrawledPage{
		PageURL:     canonicalurltest.CanonicalURLOf(t, "http://host/"),
		ContentType: "application/pdf",
	}
	if len(scene.crawledPages.indexable) != 1 || scene.crawledPages.indexable[0] != wantPage {
		t.Fatalf("want the page published as indexable once, got %v", scene.crawledPages.indexable)
	}
	if len(scene.pageVisitRecords.visitedCalls) != 1 {
		t.Fatalf("recorded visits %v, want one", scene.pageVisitRecords.visitedCalls)
	}
}

func TestAPageThatIsNotHTMLPutsNoLinksOnTheFrontier(t *testing.T) {
	scene := sceneFetching(fetchedOutcomeNotHTML())

	scene.completedVisit(t)

	if len(scene.frontier.published) != 0 {
		t.Fatalf("frontier got %v, want no link from a page that is not HTML",
			scene.frontier.published)
	}
}

func TestAPageThatIsNotHTMLCanRefuseIndexingInItsHeaders(t *testing.T) {
	fetchOutcome := fetchedOutcomeNotHTML()
	fetchOutcome.Page.RobotsTagValues = []string{"noindex"}
	scene := sceneFetching(fetchOutcome)

	scene.completedVisit(t)

	if len(scene.crawledPages.refused) != 1 || len(scene.crawledPages.indexable) != 0 {
		t.Fatalf(
			"want the page published as refusing indexing only, got refused %v, indexable %v",
			scene.crawledPages.refused, scene.crawledPages.indexable,
		)
	}
}

func TestUnreadablePageHTMLCompletesUnpublished(t *testing.T) {
	scene := sceneFetching(fetchedOutcome(pageLinkingNext))
	scene.pageReading = unreadableHTMLPage{err: errors.New("boom")}

	outcome := scene.completedVisit(t)

	if outcome.Result != yacycrawlcontract.ResultUnreadableHTML {
		t.Fatalf("result = %q, want unreadable html", outcome.Result)
	}
}

func TestAFetchThatEndsWithoutAPageConcludesByItsStatus(t *testing.T) {
	for status, want := range map[pagefetch.FetchStatus]yacycrawlcontract.PageVisitResult{
		pagefetch.FetchRejected:              yacycrawlcontract.ResultFetchRejected,
		pagefetch.FetchGone:                  yacycrawlcontract.ResultNotFound,
		pagefetch.FetchOversized:             yacycrawlcontract.ResultOversized,
		pagefetch.FetchRedirectTargetInvalid: yacycrawlcontract.ResultRedirectTargetInvalid,
		pagefetch.FetchAccessRefused:         yacycrawlcontract.ResultAccessRefused,
		pagefetch.FetchNotModified:           yacycrawlcontract.ResultNotModified,
	} {
		scene := sceneFetching(pagefetch.FetchOutcome{Status: status})

		outcome := scene.completedVisit(t)

		if outcome.Result != want {
			t.Fatalf("status %v: result = %q, want %q", status, outcome.Result, want)
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

	outcome := scene.visit(t)

	if outcome.Conclusion != pagevisit.PageVisitDropped ||
		outcome.DropReason != yacycrawlcontract.DropReasonNotDue {
		t.Fatalf("outcome = %+v, want dropped as not due", outcome)
	}
	if scene.fetch.sawFetch {
		t.Fatal("a page not due was fetched")
	}
}

func TestTheRecrawlRuleIsAskedUnderTheGraceOfTheOrder(t *testing.T) {
	scene := sceneFetching(fetchedOutcome(pageLinkingNext))
	scene.profile.RecrawlGrace = yacycrawlcontracttest.RecrawlGraceOf(t, time.Hour)

	scene.visit(t)

	if asked := scene.pageVisitRecords.askedGrace.Length(); asked != time.Hour {
		t.Fatalf("asked grace = %s, want the 1h of the order", asked)
	}
}

func TestTheFetchAsksForTheVersionTheRecrawlKnows(t *testing.T) {
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

func TestALastPageVisitThatCannotBeReadIsReportedAndThePageFetchedAsNeverVisited(t *testing.T) {
	scene := sceneFetching(fetchedOutcome(pageLinkingNext))
	unreadable := errors.New("records down")
	scene.pageVisitRecords.readErr = unreadable
	scene.pageVisitRecords.due = false
	scene.pageVisitRecords.version = pagefetch.PageVersion{EntityTag: `"known"`}

	scene.completedVisit(t)

	if !scene.fetch.sawFetch || scene.fetch.gotVersion != (pagefetch.PageVersion{}) {
		t.Fatalf("fetched %v with version %+v, want a fetch knowing no version",
			scene.fetch.sawFetch, scene.fetch.gotVersion)
	}
	if len(scene.recordFailures.notRead) != 1 ||
		!errors.Is(scene.recordFailures.notRead[0], unreadable) {
		t.Fatalf("reported %v, want the unread last page visit", scene.recordFailures.notRead)
	}
}

func TestAPageVisitThatCannotBeRecordedIsReported(t *testing.T) {
	scene := sceneFetching(fetchedOutcome(pageLinkingNext))
	unrecorded := errors.New("records down")
	scene.pageVisitRecords.recordErr = unrecorded

	scene.completedVisit(t)

	if len(scene.recordFailures.notRecorded) != 1 ||
		!errors.Is(scene.recordFailures.notRecorded[0], unrecorded) {
		t.Fatalf("reported %v, want the unrecorded page visit", scene.recordFailures.notRecorded)
	}
}
