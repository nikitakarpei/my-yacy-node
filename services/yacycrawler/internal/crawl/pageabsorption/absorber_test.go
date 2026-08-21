package pageabsorption_test

import (
	"context"
	"errors"
	"testing"

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
	a := newAbsorber(fakeExtract{err: err}, &recordingPublisher{})
	return absorb(t, a, succeeded("http://host/"))
}

func TestAbsorbPublishesTheExtractedDocument(t *testing.T) {
	publisher := &recordingPublisher{}
	a := newAbsorber(fakeExtract{document: document("t", "body")}, publisher)

	outcome := absorb(t, a, succeeded("http://host/"))

	if got := publisher.published(); len(got) != 1 || got[0].Title != "t" {
		t.Fatalf("want the document published, got %+v", got)
	}
	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("a published page carries no disposal reason, got %q", outcome.Disposal)
	}
}

func TestAbsorbPublishesUnderTheFetchedPageURL(t *testing.T) {
	publisher := &recordingPublisher{}
	a := newAbsorber(fakeExtract{document: document("t", "body")}, publisher)

	absorb(t, a, succeeded("http://HOST:80/a"))

	if got := publisher.published(); got[0].CanonicalURL != "http://host/a" {
		t.Fatalf("want the canonical page url, got %q", got[0].CanonicalURL)
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
	a := newAbsorber(fakeExtract{document: document("t", "b")}, &recordingPublisher{})

	outcome := absorb(t, a, succeeded("::not a url"))

	if outcome.Disposal != disposal.UncanonicalizableURL {
		t.Fatalf("want uncanonicalizable-url disposal, got %q", outcome.Disposal)
	}
}

func TestAbsorbHonorsMetaNoIndex(t *testing.T) {
	publisher := &recordingPublisher{}
	a := newAbsorber(fakeExtract{document: refusingDocument()}, publisher)

	outcome := absorb(t, a, succeeded("http://host/"))

	if len(publisher.published()) != 0 || outcome.Disposal != disposal.IndexingRefused {
		t.Fatalf("noindex not honored: published=%+v disposal=%q",
			publisher.published(), outcome.Disposal)
	}
}

func TestAbsorbPublishesRefusedIndexingWhenTheOrderIgnoresIt(t *testing.T) {
	publisher := &recordingPublisher{}
	a := pageabsorption.New(fakeExtract{document: refusingDocument()}, publisher, &manualClock{}).
		AbsorberFor(pageabsorption.Ignored)

	outcome := absorb(t, a, succeeded("http://host/"))

	if len(publisher.published()) != 1 || outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("noindex not ignored: published=%+v disposal=%q",
			publisher.published(), outcome.Disposal)
	}
}

func TestAbsorbHonorsNoFollow(t *testing.T) {
	a := newAbsorber(
		fakeExtract{document: linkingDocument("http://host/next")},
		&recordingPublisher{},
	)

	page := succeeded("http://host/")
	page.RefusesLinkDiscovery = true
	if outcome := absorb(t, a, page); len(outcome.DiscoveredURLs) != 0 {
		t.Fatalf("nofollow should suppress discovered links, got %v", outcome.DiscoveredURLs)
	}
}

func TestAbsorbReturnsDiscoveredLinks(t *testing.T) {
	a := newAbsorber(
		fakeExtract{document: linkingDocument("http://host/next")},
		&recordingPublisher{},
	)

	outcome := absorb(t, a, succeeded("http://host/"))
	if len(outcome.DiscoveredURLs) != 1 || outcome.DiscoveredURLs[0] != "http://host/next" {
		t.Fatalf("want the discovered link returned, got %v", outcome.DiscoveredURLs)
	}
}

func TestAbsorbReportsOversized(t *testing.T) {
	a := newAbsorber(fakeExtract{}, &recordingPublisher{})

	page := succeeded("http://host/")
	page.Truncated = true
	outcome := absorb(t, a, page)

	if outcome.Disposal != disposal.Oversized {
		t.Fatalf("want oversized disposal, got %q", outcome.Disposal)
	}
}

func TestAbsorbPublicationErrorFails(t *testing.T) {
	publisher := &recordingPublisher{failWith: errors.New("hard broker error")}
	a := newAbsorber(fakeExtract{document: document("", "b")}, publisher)

	if _, err := a.Absorb(context.Background(), succeeded("http://host/")); err == nil {
		t.Fatal("publish error should fail absorption")
	}
}
