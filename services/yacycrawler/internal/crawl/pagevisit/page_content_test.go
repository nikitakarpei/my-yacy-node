package pagevisit_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagerefusals"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

const (
	pageLinkingNext      = `<html><body><a href="/next">next</a></body></html>`
	pageRefusingIndexing = `<html><head><meta name="robots" content="noindex">` +
		`</head><body><a href="/next">next</a></body></html>`
	pageRefusingLinkDiscovery = `<html><head><meta name="robots" content="nofollow">` +
		`</head><body><a href="/next">next</a></body></html>`
)

func pageContentOutcome(t *testing.T, page pagefetch.FetchedPage) pagevisit.VisitOutcome {
	t.Helper()
	return visitHost(t, newVisitor(
		fetchOf(fetchOutcomeOf(page)),
		&fakeRecrawl{due: true},
		newObserver(),
		&fakeScrapeRequests{},
	))
}

func refusalsHonoredFor(t *testing.T, markup string) map[string]int {
	t.Helper()
	observer := newObserver()
	visitHost(t, newVisitor(
		fetchOf(fetchOutcomeOf(pageHolding(t, markup))),
		&fakeRecrawl{due: true},
		observer,
		&fakeScrapeRequests{},
	))
	return observer.refusals
}

func fetchOutcomeOf(page pagefetch.FetchedPage) pagefetch.FetchOutcome {
	return pagefetch.FetchOutcome{Status: pagefetch.FetchSucceeded, Page: page}
}

func fetchedPage(t *testing.T) pagefetch.FetchedPage {
	t.Helper()
	return pageHolding(t, pageLinkingNext)
}

func pageHolding(t *testing.T, markup string) pagefetch.FetchedPage {
	t.Helper()
	return pagefetch.FetchedPage{
		LandedURL:   canonicalurltest.CanonicalURLOf(t, "http://host/"),
		ContentType: "text/html",
		Body:        []byte(markup),
	}
}

func TestVisitLeavesAReadablePageUndisposed(t *testing.T) {
	outcome := pageContentOutcome(t, fetchedPage(t))

	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("a readable page carries no disposal reason, got %q", outcome.Disposal)
	}
}

func TestVisitReportsUnsupportedMediaType(t *testing.T) {
	page := fetchedPage(t)
	page.ContentType = "application/pdf"

	outcome := pageContentOutcome(t, page)

	if outcome.Disposal != disposal.UnsupportedMediaType {
		t.Fatalf("want unsupported-media-type disposal, got %q", outcome.Disposal)
	}
}

func TestVisitHonorsMetaNoIndex(t *testing.T) {
	outcome := pageContentOutcome(t, pageHolding(t, pageRefusingIndexing))

	if outcome.Disposal != disposal.IndexingRefused {
		t.Fatalf("noindex not honored, disposal = %q", outcome.Disposal)
	}
}

func TestVisitReportsAnHonoredIndexingRefusal(t *testing.T) {
	honored := refusalsHonoredFor(t, pageRefusingIndexing)

	if honored["indexing"] != 1 {
		t.Fatalf("honored refusals %v, want one indexing refusal", honored)
	}
}

func TestVisitReportsNoHonoredIndexingRefusalWhenTheOrderIgnoresIt(t *testing.T) {
	observer := newObserver()
	visitorFor := newVisitorFor(
		fetchOf(fetchOutcomeOf(pageHolding(t, pageRefusingIndexing))),
		&fakeRecrawl{due: true},
		observer,
		&fakeScrapeRequests{},
	)

	visitHost(t, visitorFor(pagerefusals.IgnoredRefusals{IndexingRefusal: true}))

	if observer.refusals["indexing"] != 0 {
		t.Fatalf("honored refusals %v, want none", observer.refusals)
	}
}

func TestVisitLeavesRefusedIndexingUndisposedWhenTheOrderIgnoresIt(t *testing.T) {
	visitorFor := newVisitorFor(
		fetchOf(fetchOutcomeOf(pageHolding(t, pageRefusingIndexing))),
		&fakeRecrawl{due: true},
		newObserver(),
		&fakeScrapeRequests{},
	)

	outcome := visitHost(t, visitorFor(pagerefusals.IgnoredRefusals{IndexingRefusal: true}))

	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("noindex not ignored, disposal = %q", outcome.Disposal)
	}
}

func TestVisitReportsAnHonoredLinkDiscoveryRefusal(t *testing.T) {
	honored := refusalsHonoredFor(t, pageRefusingLinkDiscovery)

	if honored["link-discovery"] != 1 {
		t.Fatalf("honored refusals %v, want one link-discovery refusal", honored)
	}
}

func TestVisitReportsDiscoveredLinks(t *testing.T) {
	outcome := pageContentOutcome(t, fetchedPage(t))

	if len(outcome.DiscoveredURLs) != 1 ||
		outcome.DiscoveredURLs[0] != canonicalurltest.CanonicalURLOf(t, "http://host/next") {
		t.Fatalf("want the discovered link returned, got %v", outcome.DiscoveredURLs)
	}
}
