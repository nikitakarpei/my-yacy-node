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

func absorbedOutcome(t *testing.T, page pagefetch.FetchedPage) pagevisit.VisitOutcome {
	t.Helper()
	return visitHost(t, newVisitor(
		fetchOf(fetchOutcomeOf(page)),
		&fakeRecrawl{due: true},
		newObserver(),
		&fakeScrapeRequests{},
	))
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
		FinalURL:    canonicalurltest.CanonicalURLOf(t, "http://host/"),
		ContentType: "text/html",
		Body:        []byte(markup),
	}
}

func TestVisitLeavesAReadablePageUndisposed(t *testing.T) {
	outcome := absorbedOutcome(t, fetchedPage(t))

	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("a readable page carries no disposal reason, got %q", outcome.Disposal)
	}
}

func TestVisitReportsUnsupportedMediaType(t *testing.T) {
	page := fetchedPage(t)
	page.ContentType = "application/pdf"

	outcome := absorbedOutcome(t, page)

	if outcome.Disposal != disposal.UnsupportedMediaType {
		t.Fatalf("want unsupported-media-type disposal, got %q", outcome.Disposal)
	}
}

func TestVisitReportsOversized(t *testing.T) {
	page := fetchedPage(t)
	page.Truncated = true

	outcome := absorbedOutcome(t, page)

	if outcome.Disposal != disposal.Oversized {
		t.Fatalf("want oversized disposal, got %q", outcome.Disposal)
	}
}

func TestVisitHonorsMetaNoIndex(t *testing.T) {
	outcome := absorbedOutcome(t, pageHolding(t, pageRefusingIndexing))

	if outcome.Disposal != disposal.IndexingRefused {
		t.Fatalf("noindex not honored, disposal = %q", outcome.Disposal)
	}
}

func TestVisitStillFollowsAPageThatRefusesIndexing(t *testing.T) {
	outcome := absorbedOutcome(t, pageHolding(t, pageRefusingIndexing))

	if len(outcome.DiscoveredURLs) != 1 {
		t.Fatalf("noindex should leave links followable, got %v", outcome.DiscoveredURLs)
	}
}

func TestVisitLeavesRefusedIndexingUndisposedWhenTheOrderIgnoresIt(t *testing.T) {
	source := newVisitorSource(
		fetchOf(fetchOutcomeOf(pageHolding(t, pageRefusingIndexing))),
		&fakeRecrawl{due: true},
		newObserver(),
		&fakeScrapeRequests{},
	)

	outcome := visitHost(t, source.VisitorFor(pagevisit.Ignored))

	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("noindex not ignored, disposal = %q", outcome.Disposal)
	}
}

func TestVisitHonorsMetaNoFollow(t *testing.T) {
	outcome := absorbedOutcome(t, pageHolding(t, pageRefusingLinkDiscovery))

	if len(outcome.DiscoveredURLs) != 0 {
		t.Fatalf("nofollow should suppress discovered links, got %v", outcome.DiscoveredURLs)
	}
}

func TestVisitHonorsARefusalTheHeadersState(t *testing.T) {
	page := fetchedPage(t)
	page.RefusesLinkDiscovery = true

	outcome := absorbedOutcome(t, page)

	if len(outcome.DiscoveredURLs) != 0 {
		t.Fatalf("nofollow should suppress discovered links, got %v", outcome.DiscoveredURLs)
	}
}

func TestVisitReportsDiscoveredLinks(t *testing.T) {
	outcome := absorbedOutcome(t, fetchedPage(t))

	if len(outcome.DiscoveredURLs) != 1 ||
		outcome.DiscoveredURLs[0] != canonicalurltest.CanonicalURLOf(t, "http://host/next") {
		t.Fatalf("want the discovered link returned, got %v", outcome.DiscoveredURLs)
	}
}
