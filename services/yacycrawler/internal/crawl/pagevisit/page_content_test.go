package pagevisit_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

const (
	pageLinkingNext      = `<html><body><a href="/next">next</a></body></html>`
	pageRefusingIndexing = `<html><head><meta name="robots" content="noindex">` +
		`</head><body><a href="/next">next</a></body></html>`
	pageRefusingLinkDiscovery = `<html><head><meta name="robots" content="nofollow">` +
		`</head><body><a href="/next">next</a></body></html>`
)

func pageContentOutcome(t *testing.T, page pagefetch.FetchedPage) pagevisit.PageVisitOutcome {
	t.Helper()
	return visitHostPage(t, newPageVisitor(
		fetchOf(fetchOutcomeOf(page)),
		&fakePageVisits{due: true},
		newObserver(),
		&fakeCrawledPages{},
	))
}

func linkDiscoveryRefusalsEnforcedFor(t *testing.T, markup string) int {
	t.Helper()
	observer := newObserver()
	visitHostPage(t, newPageVisitor(
		fetchOf(fetchOutcomeOf(pageHolding(t, markup))),
		&fakePageVisits{due: true},
		observer,
		&fakeCrawledPages{},
	))
	return observer.linkDiscoveryRefusalsEnforced
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

func TestVisitPublishesAPageThatRefusesIndexingAsRefusingIndexing(t *testing.T) {
	crawledPages := &fakeCrawledPages{}
	pageVisitor := newPageVisitor(
		fetchOf(fetchOutcomeOf(pageHolding(t, pageRefusingIndexing))),
		&fakePageVisits{due: true},
		newObserver(),
		crawledPages,
	)

	outcome := visitHostPage(t, pageVisitor)

	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("a published page carries no disposal, got %q", outcome.Disposal)
	}
	if refused := crawledPages.refusedPages(); len(refused) != 1 {
		t.Fatalf("want the page published as refusing indexing, got %v", refused)
	}
	if indexable := crawledPages.indexablePages(); len(indexable) != 0 {
		t.Fatalf("a page that refuses indexing is not indexable, got %v", indexable)
	}
}

func TestVisitReportsAnHonoredLinkDiscoveryRefusal(t *testing.T) {
	enforced := linkDiscoveryRefusalsEnforcedFor(t, pageRefusingLinkDiscovery)

	if enforced != 1 {
		t.Fatalf("link discovery refusals enforced = %d, want 1", enforced)
	}
}

func TestVisitReportsDiscoveredLinks(t *testing.T) {
	outcome := pageContentOutcome(t, fetchedPage(t))

	if len(outcome.DiscoveredURLs) != 1 ||
		outcome.DiscoveredURLs[0] != canonicalurltest.CanonicalURLOf(t, "http://host/next") {
		t.Fatalf("want the discovered link returned, got %v", outcome.DiscoveredURLs)
	}
}
