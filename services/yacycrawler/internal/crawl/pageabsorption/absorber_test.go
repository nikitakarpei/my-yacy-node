package pageabsorption

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchedpage"
)

func absorb(t *testing.T, a *Absorber, page fetchedpage.Page) AbsorptionOutcome {
	t.Helper()
	outcome, err := a.Absorb(context.Background(), page)
	if err != nil {
		t.Fatalf("absorb: %v", err)
	}
	return outcome
}

func absorbWithExtractionError(t *testing.T, err error) AbsorptionOutcome {
	t.Helper()
	a := newAbsorber(fakeExtract{err: err}, &recordingPublisher{})
	return absorb(t, a, succeeded("http://host/"))
}

func TestAbsorbPublishesEveryExtractedDocument(t *testing.T) {
	extract := fakeExtract{
		documents: []contentextraction.ExtractedDocument{document("http://host/", "t", "body")},
	}
	publisher := &recordingPublisher{}
	a := newAbsorber(extract, publisher)

	outcome := absorb(t, a, succeeded("http://host/"))

	if got := publisher.published(); len(got) != 1 || got[0].Title != "t" {
		t.Fatalf("want the document published, got %+v", got)
	}
	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("a published page carries no disposal reason, got %q", outcome.Disposal)
	}
}

func TestAbsorbReportsUnsupportedMediaType(t *testing.T) {
	outcome := absorbWithExtractionError(t, contentextraction.ErrUnsupportedMediaType)

	if outcome.Disposal != disposal.UnsupportedMediaType {
		t.Fatalf("want unsupported-media-type disposal, got %q", outcome.Disposal)
	}
}

func TestAbsorbReportsNestingTooDeep(t *testing.T) {
	outcome := absorbWithExtractionError(t, contentextraction.ErrNestingTooDeep)

	if outcome.Disposal != disposal.NestingTooDeep {
		t.Fatalf("want nesting-too-deep disposal, got %q", outcome.Disposal)
	}
}

func TestAbsorbReportsDocumentBudgetExhausted(t *testing.T) {
	outcome := absorbWithExtractionError(t, contentextraction.ErrDocumentBudgetExhausted)

	if outcome.Disposal != disposal.DocumentBudgetExhausted {
		t.Fatalf("want document-budget-exhausted disposal, got %q", outcome.Disposal)
	}
}

func TestAbsorbReportsUnextractableOnUnknownExtractionError(t *testing.T) {
	outcome := absorbWithExtractionError(t, errors.New("parser broke"))

	if outcome.Disposal != disposal.Unextractable {
		t.Fatalf("want unextractable disposal, got %q", outcome.Disposal)
	}
}

func TestAbsorbReportsEmptyExtraction(t *testing.T) {
	a := newAbsorber(fakeExtract{documents: nil}, &recordingPublisher{})

	outcome := absorb(t, a, succeeded("http://host/"))

	if outcome.Disposal != disposal.Unextractable {
		t.Fatalf("want unextractable disposal, got %q", outcome.Disposal)
	}
}

func TestAbsorbReportsUncanonicalizableDocumentURL(t *testing.T) {
	extract := fakeExtract{
		documents: []contentextraction.ExtractedDocument{document("::not a url", "t", "b")},
	}
	a := newAbsorber(extract, &recordingPublisher{})

	outcome := absorb(t, a, succeeded("http://host/"))

	if outcome.Disposal != disposal.UncanonicalizableURL {
		t.Fatalf("want uncanonicalizable-url disposal, got %q", outcome.Disposal)
	}
}

func TestAbsorbFansOutContainerDocuments(t *testing.T) {
	extract := fakeExtract{documents: []contentextraction.ExtractedDocument{
		document("http://host/a.zip!/one.html", "one", "a"),
		document("http://host/a.zip!/two.html", "two", "b"),
	}}
	publisher := &recordingPublisher{}
	a := newAbsorber(extract, publisher)

	page := succeeded("http://host/a.zip")
	page.ContentType = "application/zip"
	absorb(t, a, page)

	got := publisher.published()
	if len(got) != 2 {
		t.Fatalf("want 2 member documents published, got %+v", got)
	}
	if got[0].CanonicalURL == got[1].CanonicalURL {
		t.Fatalf("members collapsed to one URL: %+v", got)
	}
}

func TestAbsorbReportsNoDisposalWhenOneMemberPublishes(t *testing.T) {
	extract := fakeExtract{documents: []contentextraction.ExtractedDocument{
		{URL: "http://host/a.zip!/one.html", ExtractedContent: contentextraction.ExtractedContent{
			Body:            []byte("a"),
			Format:          contentformatgraph.FormatReadableText,
			RefusesIndexing: true,
		}},
		document("http://host/a.zip!/two.html", "two", "b"),
	}}
	a := newAbsorber(extract, &recordingPublisher{})

	outcome := absorb(t, a, succeeded("http://host/a.zip"))

	if outcome.Disposal != disposal.NotDisposed {
		t.Fatalf("one published member leaves the page published, got %+v", outcome)
	}
}

func TestAbsorbHonorsMetaNoIndex(t *testing.T) {
	extract := fakeExtract{documents: []contentextraction.ExtractedDocument{{
		URL: "http://host/",
		ExtractedContent: contentextraction.ExtractedContent{
			Body:            []byte("b"),
			Format:          contentformatgraph.FormatReadableText,
			RefusesIndexing: true,
		},
	}}}
	publisher := &recordingPublisher{}
	a := newAbsorber(extract, publisher)

	outcome := absorb(t, a, succeeded("http://host/"))

	if len(publisher.published()) != 0 || outcome.Disposal != disposal.IndexingRefused {
		t.Fatalf("noindex not honored: published=%+v disposal=%q",
			publisher.published(), outcome.Disposal)
	}
}

func TestAbsorbHonorsNoFollow(t *testing.T) {
	extract := fakeExtract{documents: []contentextraction.ExtractedDocument{{
		URL: "http://host/",
		ExtractedContent: contentextraction.ExtractedContent{
			Body:           []byte("b"),
			Format:         contentformatgraph.FormatReadableText,
			DiscoveredURLs: []string{"http://host/next"},
		},
	}}}
	a := newAbsorber(extract, &recordingPublisher{})

	page := succeeded("http://host/")
	page.RefusesLinkDiscovery = true
	if outcome := absorb(t, a, page); len(outcome.DiscoveredURLs) != 0 {
		t.Fatalf("nofollow should suppress discovered links, got %v", outcome.DiscoveredURLs)
	}
}

func TestAbsorbReturnsDiscoveredLinks(t *testing.T) {
	extract := fakeExtract{documents: []contentextraction.ExtractedDocument{{
		URL: "http://host/",
		ExtractedContent: contentextraction.ExtractedContent{
			Body:           []byte("b"),
			Format:         contentformatgraph.FormatReadableText,
			DiscoveredURLs: []string{"http://host/next"},
		},
	}}}
	a := newAbsorber(extract, &recordingPublisher{})

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
	extract := fakeExtract{
		documents: []contentextraction.ExtractedDocument{document("http://host/", "", "b")},
	}
	publisher := &recordingPublisher{failWith: errors.New("hard broker error")}
	a := newAbsorber(extract, publisher)

	if _, err := a.Absorb(context.Background(), succeeded("http://host/")); err == nil {
		t.Fatal("publish error should fail absorption")
	}
}
