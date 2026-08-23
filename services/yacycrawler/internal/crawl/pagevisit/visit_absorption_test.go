package pagevisit_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagelinks"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

func absorbedOutcome(
	t *testing.T,
	pageLinks fakePageLinks,
	page pagefetch.FetchedPage,
) pagevisit.VisitOutcome {
	t.Helper()
	return visitHost(t, newVisitor(
		fetchOf(fetchOutcomeOf(page)),
		&fakeRecrawl{due: true},
		pageLinks,
		newObserver(),
		&fakeScrapeRequests{},
	))
}

func fetchOutcomeOf(page pagefetch.FetchedPage) pagefetch.FetchOutcome {
	return pagefetch.FetchOutcome{Status: pagefetch.FetchSucceeded, Page: page}
}

func fetchedPage(t *testing.T) pagefetch.FetchedPage {
	t.Helper()
	return pagefetch.FetchedPage{
		FinalURL:    canonicalurltest.CanonicalURLOf(t, "http://host/"),
		ContentType: "text/html",
		Body:        []byte("x"),
	}
}

func TestVisitLeavesAReadablePageUndisposed(t *testing.T) {
	outcome := absorbedOutcome(t, readablePageLinks(), fetchedPage(t))

	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("a readable page carries no disposal reason, got %q", outcome.Disposal)
	}
}

func TestVisitReportsUnsupportedMediaType(t *testing.T) {
	outcome := absorbedOutcome(
		t,
		fakePageLinks{err: pagelinks.ErrNotHTML},
		fetchedPage(t),
	)

	if outcome.Disposal != disposal.UnsupportedMediaType {
		t.Fatalf("want unsupported-media-type disposal, got %q", outcome.Disposal)
	}
}

func TestVisitReportsUnextractableOnUnknownExtractionError(t *testing.T) {
	outcome := absorbedOutcome(
		t,
		fakePageLinks{err: errors.New("parser broke")},
		fetchedPage(t),
	)

	if outcome.Disposal != disposal.Unextractable {
		t.Fatalf("want unextractable disposal, got %q", outcome.Disposal)
	}
}

func TestVisitReportsOversized(t *testing.T) {
	page := fetchedPage(t)
	page.Truncated = true

	outcome := absorbedOutcome(t, readablePageLinks(), page)

	if outcome.Disposal != disposal.Oversized {
		t.Fatalf("want oversized disposal, got %q", outcome.Disposal)
	}
}

func TestVisitHonorsMetaNoIndex(t *testing.T) {
	outcome := absorbedOutcome(
		t,
		fakePageLinks{links: refusingLinks()},
		fetchedPage(t),
	)

	if outcome.Disposal != disposal.IndexingRefused {
		t.Fatalf("noindex not honored, disposal = %q", outcome.Disposal)
	}
}

func TestVisitLeavesRefusedIndexingUndisposedWhenTheOrderIgnoresIt(t *testing.T) {
	source := newVisitorSource(
		fetchOf(fetchOutcomeOf(fetchedPage(t))),
		&fakeRecrawl{due: true},
		fakePageLinks{links: refusingLinks()},
		newObserver(),
		&fakeScrapeRequests{},
	)

	outcome := visitHost(t, source.VisitorFor(pagevisit.Ignored))

	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("noindex not ignored, disposal = %q", outcome.Disposal)
	}
}

func TestVisitHonorsNoFollow(t *testing.T) {
	page := fetchedPage(t)
	page.RefusesLinkDiscovery = true

	outcome := absorbedOutcome(
		t,
		fakePageLinks{links: linkingLinks(t, "http://host/next")},
		page,
	)

	if len(outcome.DiscoveredURLs) != 0 {
		t.Fatalf("nofollow should suppress discovered links, got %v", outcome.DiscoveredURLs)
	}
}

func TestVisitReportsDiscoveredLinks(t *testing.T) {
	outcome := absorbedOutcome(
		t,
		fakePageLinks{links: linkingLinks(t, "http://host/next")},
		fetchedPage(t),
	)

	if len(outcome.DiscoveredURLs) != 1 ||
		outcome.DiscoveredURLs[0] != canonicalurltest.CanonicalURLOf(t, "http://host/next") {
		t.Fatalf("want the discovered link returned, got %v", outcome.DiscoveredURLs)
	}
}
