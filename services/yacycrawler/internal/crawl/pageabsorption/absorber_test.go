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

func TestAbsorbPublishesEveryExtractedDocument(t *testing.T) {
	extract := fakeExtract{
		documents: []contentextraction.ExtractedDocument{document("http://host/", "t", "body")},
	}
	publisher := &recordingPublisher{}
	a := newAbsorber(extract, publisher, newObserver())

	outcome := absorb(t, a, succeeded("http://host/"))

	if got := publisher.published(); len(got) != 1 || got[0].Title != "t" {
		t.Fatalf("want the document published, got %+v", got)
	}
	if !outcome.Published {
		t.Fatal("want outcome to report published")
	}
}

func TestAbsorbDisposesUnsupportedMediaType(t *testing.T) {
	observer := newObserver()
	a := newAbsorber(
		fakeExtract{err: contentextraction.ErrUnsupportedMediaType},
		&recordingPublisher{}, observer,
	)

	absorb(t, a, succeeded("http://host/"))

	if observer.disposed[disposal.UnsupportedMediaType] != 1 {
		t.Fatalf("want unsupported-media-type disposal, got %v", observer.disposed)
	}
}

func TestAbsorbDisposesNestingTooDeep(t *testing.T) {
	observer := newObserver()
	a := newAbsorber(
		fakeExtract{err: contentextraction.ErrNestingTooDeep},
		&recordingPublisher{}, observer,
	)

	absorb(t, a, succeeded("http://host/"))

	if observer.disposed[disposal.NestingTooDeep] != 1 {
		t.Fatalf("want nesting-too-deep disposal, got %v", observer.disposed)
	}
}

func TestAbsorbDisposesDocumentBudgetExhausted(t *testing.T) {
	observer := newObserver()
	a := newAbsorber(
		fakeExtract{err: contentextraction.ErrDocumentBudgetExhausted},
		&recordingPublisher{}, observer,
	)

	absorb(t, a, succeeded("http://host/"))

	if observer.disposed[disposal.DocumentBudgetExhausted] != 1 {
		t.Fatalf("want document-budget-exhausted disposal, got %v", observer.disposed)
	}
}

func TestAbsorbDisposesEmptyExtraction(t *testing.T) {
	observer := newObserver()
	a := newAbsorber(fakeExtract{documents: nil}, &recordingPublisher{}, observer)

	absorb(t, a, succeeded("http://host/"))

	if observer.disposed[disposal.Unextractable] != 1 {
		t.Fatalf("want unextractable disposal, got %v", observer.disposed)
	}
}

func TestAbsorbFansOutContainerDocuments(t *testing.T) {
	extract := fakeExtract{documents: []contentextraction.ExtractedDocument{
		document("http://host/a.zip!/one.html", "one", "a"),
		document("http://host/a.zip!/two.html", "two", "b"),
	}}
	publisher := &recordingPublisher{}
	a := newAbsorber(extract, publisher, newObserver())

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
	observer := newObserver()
	a := newAbsorber(extract, publisher, observer)

	absorb(t, a, succeeded("http://host/"))

	if len(publisher.published()) != 0 || observer.disposed[disposal.IndexingRefused] != 1 {
		t.Fatalf("noindex not honored: published=%+v disposed=%v",
			publisher.published(), observer.disposed)
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
	a := newAbsorber(extract, &recordingPublisher{}, newObserver())

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
	a := newAbsorber(extract, &recordingPublisher{}, newObserver())

	outcome := absorb(t, a, succeeded("http://host/"))
	if len(outcome.DiscoveredURLs) != 1 || outcome.DiscoveredURLs[0] != "http://host/next" {
		t.Fatalf("want the discovered link returned, got %v", outcome.DiscoveredURLs)
	}
}

func TestAbsorbDisposesOversized(t *testing.T) {
	observer := newObserver()
	a := newAbsorber(fakeExtract{}, &recordingPublisher{}, observer)

	page := succeeded("http://host/")
	page.Truncated = true
	absorb(t, a, page)

	if observer.disposed[disposal.Oversized] != 1 {
		t.Fatalf("want oversized disposal, got %v", observer.disposed)
	}
}

func TestAbsorbPublicationErrorFails(t *testing.T) {
	extract := fakeExtract{
		documents: []contentextraction.ExtractedDocument{document("http://host/", "", "b")},
	}
	publisher := &recordingPublisher{failWith: errors.New("hard broker error")}
	a := newAbsorber(extract, publisher, newObserver())

	if _, err := a.Absorb(context.Background(), succeeded("http://host/")); err == nil {
		t.Fatal("publish error should fail absorption")
	}
}
