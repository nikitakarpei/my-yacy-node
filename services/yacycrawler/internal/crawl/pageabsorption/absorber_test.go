package pageabsorption_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
)

func absorb(
	t *testing.T,
	a pageabsorption.Absorber,
	page pagefetch.FetchedPage,
) pageabsorption.AbsorptionOutcome {
	t.Helper()
	outcome, err := a.Absorb(context.Background(), page)
	if err != nil {
		t.Fatalf("absorb: %v", err)
	}
	return outcome
}

func absorbWithExtractionError(t *testing.T, err error) pageabsorption.AbsorptionOutcome {
	t.Helper()
	return absorb(t, newAbsorber(fakeExtract{err: err}), succeeded("http://host/"))
}

func TestAbsorbLeavesAReadablePageUndisposed(t *testing.T) {
	a := newAbsorber(fakeExtract{document: document("t", "body")})

	outcome := absorb(t, a, succeeded("http://host/"))

	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("a readable page carries no disposal reason, got %q", outcome.Disposal)
	}
}

func TestAbsorbReportsUnsupportedMediaType(t *testing.T) {
	outcome := absorbWithExtractionError(t, contentextraction.ErrUnsupportedMediaType)

	if outcome.Disposal != disposal.UnsupportedMediaType {
		t.Fatalf("want unsupported-media-type disposal, got %q", outcome.Disposal)
	}
}

func TestAbsorbReportsUnextractableOnUnknownExtractionError(t *testing.T) {
	outcome := absorbWithExtractionError(t, errors.New("parser broke"))

	if outcome.Disposal != disposal.Unextractable {
		t.Fatalf("want unextractable disposal, got %q", outcome.Disposal)
	}
}

func TestAbsorbReportsUncanonicalizablePageURL(t *testing.T) {
	a := newAbsorber(fakeExtract{document: document("t", "b")})

	outcome := absorb(t, a, succeeded("::not a url"))

	if outcome.Disposal != disposal.UncanonicalizableURL {
		t.Fatalf("want uncanonicalizable-url disposal, got %q", outcome.Disposal)
	}
}

func TestAbsorbHonorsMetaNoIndex(t *testing.T) {
	a := newAbsorber(fakeExtract{document: refusingDocument()})

	outcome := absorb(t, a, succeeded("http://host/"))

	if outcome.Disposal != disposal.IndexingRefused {
		t.Fatalf("noindex not honored, disposal = %q", outcome.Disposal)
	}
}

func TestAbsorbLeavesRefusedIndexingUndisposedWhenTheOrderIgnoresIt(t *testing.T) {
	a := pageabsorption.New(fakeExtract{document: refusingDocument()}).
		AbsorberFor(pageabsorption.Ignored)

	outcome := absorb(t, a, succeeded("http://host/"))

	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("noindex not ignored, disposal = %q", outcome.Disposal)
	}
}

func TestAbsorbHonorsNoFollow(t *testing.T) {
	a := newAbsorber(fakeExtract{document: linkingDocument(t, "http://host/next")})

	page := succeeded("http://host/")
	page.RefusesLinkDiscovery = true
	if outcome := absorb(t, a, page); len(outcome.DiscoveredURLs) != 0 {
		t.Fatalf("nofollow should suppress discovered links, got %v", outcome.DiscoveredURLs)
	}
}

func TestAbsorbReturnsDiscoveredLinks(t *testing.T) {
	a := newAbsorber(fakeExtract{document: linkingDocument(t, "http://host/next")})

	outcome := absorb(t, a, succeeded("http://host/"))
	if len(outcome.DiscoveredURLs) != 1 ||
		outcome.DiscoveredURLs[0] != canonicalurltest.CanonicalURLOf(t, "http://host/next") {
		t.Fatalf("want the discovered link returned, got %v", outcome.DiscoveredURLs)
	}
}

func TestAbsorbReportsOversized(t *testing.T) {
	a := newAbsorber(fakeExtract{})

	page := succeeded("http://host/")
	page.Truncated = true
	outcome := absorb(t, a, page)

	if outcome.Disposal != disposal.Oversized {
		t.Fatalf("want oversized disposal, got %q", outcome.Disposal)
	}
}
