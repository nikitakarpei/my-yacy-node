package pagevisit_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

func absorbedOutcome(
	t *testing.T,
	extractor fakeExtract,
	page pagefetch.FetchedPage,
) pagevisit.VisitOutcome {
	t.Helper()
	return visitHost(t, newVisitor(
		fetchOf(fetchOutcomeOf(page)),
		&fakeRecrawl{due: true},
		extractor,
		newObserver(),
		&fakeReachedPages{},
	))
}

func fetchOutcomeOf(page pagefetch.FetchedPage) pagefetch.FetchOutcome {
	return pagefetch.FetchOutcome{Status: pagefetch.FetchSucceeded, Page: page}
}

func fetchedPage(finalURL string) pagefetch.FetchedPage {
	return pagefetch.FetchedPage{
		FinalURL:    finalURL,
		ContentType: "text/html",
		Body:        []byte("x"),
	}
}

func TestVisitLeavesAReadablePageUndisposed(t *testing.T) {
	outcome := absorbedOutcome(t, readableExtract(), fetchedPage("http://host/"))

	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("a readable page carries no disposal reason, got %q", outcome.Disposal)
	}
}

func TestVisitReportsUnsupportedMediaType(t *testing.T) {
	outcome := absorbedOutcome(
		t,
		fakeExtract{err: contentextraction.ErrUnsupportedMediaType},
		fetchedPage("http://host/"),
	)

	if outcome.Disposal != disposal.UnsupportedMediaType {
		t.Fatalf("want unsupported-media-type disposal, got %q", outcome.Disposal)
	}
}

func TestVisitReportsUnextractableOnUnknownExtractionError(t *testing.T) {
	outcome := absorbedOutcome(
		t,
		fakeExtract{err: errors.New("parser broke")},
		fetchedPage("http://host/"),
	)

	if outcome.Disposal != disposal.Unextractable {
		t.Fatalf("want unextractable disposal, got %q", outcome.Disposal)
	}
}

func TestVisitReportsUncanonicalizablePageURL(t *testing.T) {
	outcome := absorbedOutcome(t, readableExtract(), fetchedPage("::not a url"))

	if outcome.Disposal != disposal.UncanonicalizableURL {
		t.Fatalf("want uncanonicalizable-url disposal, got %q", outcome.Disposal)
	}
}

func TestVisitReportsOversized(t *testing.T) {
	page := fetchedPage("http://host/")
	page.Truncated = true

	outcome := absorbedOutcome(t, readableExtract(), page)

	if outcome.Disposal != disposal.Oversized {
		t.Fatalf("want oversized disposal, got %q", outcome.Disposal)
	}
}

func TestVisitHonorsMetaNoIndex(t *testing.T) {
	outcome := absorbedOutcome(
		t,
		fakeExtract{document: refusingDocument()},
		fetchedPage("http://host/"),
	)

	if outcome.Disposal != disposal.IndexingRefused {
		t.Fatalf("noindex not honored, disposal = %q", outcome.Disposal)
	}
}

func TestVisitLeavesRefusedIndexingUndisposedWhenTheOrderIgnoresIt(t *testing.T) {
	source := newVisitorSource(
		fetchOf(fetchOutcomeOf(fetchedPage("http://host/"))),
		&fakeRecrawl{due: true},
		fakeExtract{document: refusingDocument()},
		newObserver(),
		&fakeReachedPages{},
	)

	outcome := visitHost(t, source.VisitorFor(pagevisit.Ignored))

	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("noindex not ignored, disposal = %q", outcome.Disposal)
	}
}

func TestVisitHonorsNoFollow(t *testing.T) {
	page := fetchedPage("http://host/")
	page.RefusesLinkDiscovery = true

	outcome := absorbedOutcome(
		t,
		fakeExtract{document: linkingDocument(t, "http://host/next")},
		page,
	)

	if len(outcome.DiscoveredURLs) != 0 {
		t.Fatalf("nofollow should suppress discovered links, got %v", outcome.DiscoveredURLs)
	}
}

func TestVisitReportsDiscoveredLinks(t *testing.T) {
	outcome := absorbedOutcome(
		t,
		fakeExtract{document: linkingDocument(t, "http://host/next")},
		fetchedPage("http://host/"),
	)

	if len(outcome.DiscoveredURLs) != 1 ||
		outcome.DiscoveredURLs[0] != canonicalurltest.CanonicalURLOf(t, "http://host/next") {
		t.Fatalf("want the discovered link returned, got %v", outcome.DiscoveredURLs)
	}
}
