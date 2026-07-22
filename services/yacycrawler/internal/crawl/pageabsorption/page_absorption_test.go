package pageabsorption

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

func absorb(t *testing.T, a *Absorption, outcome pagevisit.FetchOutcome) []string {
	t.Helper()
	links, err := a.Absorb(context.Background(), outcome)
	if err != nil {
		t.Fatalf("absorb: %v", err)
	}
	return links
}

func TestAbsorbPublishesToEveryFeed(t *testing.T) {
	extract := fakeExtract{
		documents: []contentextraction.ExtractedDocument{document("http://host/", "t", "body")},
	}
	rwi := &fakeFeed{representation: yacycrawlcontract.PageRepresentationKindRWI}
	text := &fakeFeed{representation: yacycrawlcontract.PageRepresentationKindText}
	a := newAbsorption(extract, &recordingResolve{}, feeds(rwi, text), newObserver())

	absorb(t, a, succeeded("http://host/"))

	if len(rwi.published) != 1 || len(text.published) != 1 {
		t.Fatalf("feeds not both advanced: rwi=%v text=%v", rwi.published, text.published)
	}
}

func TestAbsorbRecordsRedirectEdgePerNonFinalHop(t *testing.T) {
	extract := fakeExtract{
		documents: []contentextraction.ExtractedDocument{document("http://host/c", "t", "body")},
	}
	resolve := &recordingResolve{}
	rwi := &fakeFeed{representation: yacycrawlcontract.PageRepresentationKindRWI}
	a := newAbsorption(extract, resolve, feeds(rwi), newObserver())

	outcome := succeeded("http://host/c")
	outcome.RedirectChain = []string{"http://host/a", "http://host/b", "http://host/c"}
	absorb(t, a, outcome)

	want := []redirectEdge{
		{requested: "http://host/a", canonical: "http://host/c"},
		{requested: "http://host/b", canonical: "http://host/c"},
	}
	got := resolve.recorded()
	if len(got) != len(want) {
		t.Fatalf("edges = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("edge[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestAbsorbRecordsNoRedirectEdgeOnDirectFetch(t *testing.T) {
	extract := fakeExtract{
		documents: []contentextraction.ExtractedDocument{document("http://host/", "t", "body")},
	}
	resolve := &recordingResolve{}
	rwi := &fakeFeed{representation: yacycrawlcontract.PageRepresentationKindRWI}
	a := newAbsorption(extract, resolve, feeds(rwi), newObserver())

	outcome := succeeded("http://host/")
	outcome.RedirectChain = []string{"http://host/"}
	absorb(t, a, outcome)

	if got := resolve.recorded(); len(got) != 0 {
		t.Fatalf("direct fetch recorded edges: %v", got)
	}
}

func TestAbsorbSkipsFeedRefusingPageFormat(t *testing.T) {
	extract := fakeExtract{
		documents: []contentextraction.ExtractedDocument{document("http://host/", "t", "body")},
	}
	rwi := &fakeFeed{representation: yacycrawlcontract.PageRepresentationKindRWI}
	markdown := &fakeFeed{
		representation: yacycrawlcontract.PageRepresentationKindMarkdown,
		contentFormat:  contentformatgraph.FormatMarkdown,
	}
	observer := newObserver()
	a := newAbsorption(extract, &recordingResolve{}, feeds(rwi, markdown), observer)

	absorb(t, a, succeeded("http://host/"))

	if len(rwi.published) != 1 {
		t.Fatalf("accepting feed not advanced: rwi=%v", rwi.published)
	}
	if len(markdown.published) != 0 {
		t.Fatalf("refusing feed advanced: markdown=%v", markdown.published)
	}
	if observer.disposed[DisposalUnrepresentable] != 0 {
		t.Fatalf("page disposed despite an accepting feed: %v", observer.disposed)
	}
}

func TestAbsorbDisposesPageNoFeedAccepts(t *testing.T) {
	extract := fakeExtract{
		documents: []contentextraction.ExtractedDocument{document("http://host/", "t", "body")},
	}
	rwi := &fakeFeed{
		representation: yacycrawlcontract.PageRepresentationKindRWI,
		contentFormat:  contentformatgraph.FormatMarkdown,
	}
	observer := newObserver()
	a := newAbsorption(extract, &recordingResolve{}, feeds(rwi), observer)

	absorb(t, a, succeeded("http://host/"))

	if len(rwi.published) != 0 {
		t.Fatalf("refusing feed advanced: rwi=%v", rwi.published)
	}
	if observer.disposed[DisposalUnrepresentable] != 1 {
		t.Fatalf("want unrepresentable disposal, got %v", observer.disposed)
	}
}

func TestAbsorbDisposesUnsupportedMediaType(t *testing.T) {
	observer := newObserver()
	a := newAbsorption(fakeExtract{err: contentextraction.ErrUnsupportedMediaType},
		&recordingResolve{},
		feeds(&fakeFeed{representation: yacycrawlcontract.PageRepresentationKindRWI}), observer)

	absorb(t, a, succeeded("http://host/"))

	if observer.disposed[DisposalUnsupportedMediaType] != 1 {
		t.Fatalf("want unsupported-media-type disposal, got %v", observer.disposed)
	}
}

func TestAbsorbDisposesContainerOverflow(t *testing.T) {
	observer := newObserver()
	a := newAbsorption(fakeExtract{err: contentextraction.ErrContainerOverflow},
		&recordingResolve{},
		feeds(&fakeFeed{representation: yacycrawlcontract.PageRepresentationKindRWI}), observer)

	absorb(t, a, succeeded("http://host/"))

	if observer.disposed[DisposalContainerOverflow] != 1 {
		t.Fatalf("want container-overflow disposal, got %v", observer.disposed)
	}
}

func TestAbsorbDisposesEmptyExtraction(t *testing.T) {
	observer := newObserver()
	a := newAbsorption(fakeExtract{documents: nil}, &recordingResolve{},
		feeds(&fakeFeed{representation: yacycrawlcontract.PageRepresentationKindRWI}), observer)

	absorb(t, a, succeeded("http://host/"))

	if observer.disposed[DisposalUnextractable] != 1 {
		t.Fatalf("want unextractable disposal, got %v", observer.disposed)
	}
}

func TestAbsorbFansOutContainerDocuments(t *testing.T) {
	extract := fakeExtract{documents: []contentextraction.ExtractedDocument{
		document("http://host/a.zip!/one.html", "one", "a"),
		document("http://host/a.zip!/two.html", "two", "b"),
	}}
	rwi := &fakeFeed{representation: yacycrawlcontract.PageRepresentationKindRWI}
	a := newAbsorption(extract, &recordingResolve{}, feeds(rwi), newObserver())

	outcome := succeeded("http://host/a.zip")
	outcome.ContentType = "application/zip"
	absorb(t, a, outcome)

	if len(rwi.published) != 2 {
		t.Fatalf("want 2 member documents published, got %v", rwi.published)
	}
	if rwi.published[0] == rwi.published[1] {
		t.Fatalf("members collapsed to one URL: %v", rwi.published)
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
	rwi := &fakeFeed{representation: yacycrawlcontract.PageRepresentationKindRWI}
	observer := newObserver()
	a := newAbsorption(extract, &recordingResolve{}, feeds(rwi), observer)

	absorb(t, a, succeeded("http://host/"))

	if len(rwi.published) != 0 || observer.disposed[DisposalNoIndex] != 1 {
		t.Fatalf("noindex not honored: published=%v disposed=%v", rwi.published, observer.disposed)
	}
}

func TestAbsorbHonorsNoFollow(t *testing.T) {
	extract := fakeExtract{documents: []contentextraction.ExtractedDocument{{
		URL: "http://host/",
		ExtractedContent: contentextraction.ExtractedContent{
			Body:   []byte("b"),
			Format: contentformatgraph.FormatReadableText,
			Links:  []string{"http://host/next"},
		},
	}}}
	rwi := &fakeFeed{representation: yacycrawlcontract.PageRepresentationKindRWI}
	a := newAbsorption(extract, &recordingResolve{}, feeds(rwi), newObserver())

	outcome := succeeded("http://host/")
	outcome.RefusesLinkDiscovery = true
	if links := absorb(t, a, outcome); len(links) != 0 {
		t.Fatalf("nofollow should suppress discovered links, got %v", links)
	}
}

func TestAbsorbReturnsDiscoveredLinks(t *testing.T) {
	extract := fakeExtract{documents: []contentextraction.ExtractedDocument{{
		URL: "http://host/",
		ExtractedContent: contentextraction.ExtractedContent{
			Body:   []byte("b"),
			Format: contentformatgraph.FormatReadableText,
			Links:  []string{"http://host/next"},
		},
	}}}
	rwi := &fakeFeed{representation: yacycrawlcontract.PageRepresentationKindRWI}
	a := newAbsorption(extract, &recordingResolve{}, feeds(rwi), newObserver())

	links := absorb(t, a, succeeded("http://host/"))
	if len(links) != 1 || links[0] != "http://host/next" {
		t.Fatalf("want the discovered link returned, got %v", links)
	}
}

func TestAbsorbDisposesOversized(t *testing.T) {
	observer := newObserver()
	a := newAbsorption(fakeExtract{}, &recordingResolve{},
		feeds(&fakeFeed{representation: yacycrawlcontract.PageRepresentationKindRWI}), observer)

	outcome := succeeded("http://host/")
	outcome.Truncated = true
	absorb(t, a, outcome)

	if observer.disposed[DisposalOversized] != 1 {
		t.Fatalf("want oversized disposal, got %v", observer.disposed)
	}
}

func TestAbsorbPublicationHardErrorFails(t *testing.T) {
	extract := fakeExtract{
		documents: []contentextraction.ExtractedDocument{document("http://host/", "", "b")},
	}
	feed := &fakeFeed{
		representation: yacycrawlcontract.PageRepresentationKindRWI,
		failWith:       errors.New("hard broker error"),
	}
	a := newAbsorption(extract, &recordingResolve{}, feeds(feed), newObserver())

	if _, err := a.Absorb(context.Background(), succeeded("http://host/")); err == nil {
		t.Fatal("hard publish error should fail absorption")
	}
}

func TestAbsorbRetriesTransientPublication(t *testing.T) {
	extract := fakeExtract{
		documents: []contentextraction.ExtractedDocument{document("http://host/", "", "b")},
	}
	feed := &flakyFeed{failuresLeft: 2}
	a := newAbsorption(extract, &recordingResolve{},
		[]Feed{feed}, newObserver())

	absorb(t, a, succeeded("http://host/"))

	if feed.published != 1 {
		t.Fatalf("transient publish should retry then succeed: published=%d", feed.published)
	}
}

type flakyFeed struct {
	failuresLeft int
	published    int
}

func (*flakyFeed) Representation() yacycrawlcontract.PageRepresentationKind {
	return yacycrawlcontract.PageRepresentationKindRWI
}

func (*flakyFeed) ContentFormat() contentformatgraph.Format {
	return contentformatgraph.FormatReadableText
}

func (*flakyFeed) Frame(
	CrawledPage,
	[]byte,
) (PagePublication, error) {
	return NewPagePublication(), nil
}

func (o *flakyFeed) Publish(context.Context, PagePublication) error {
	if o.failuresLeft > 0 {
		o.failuresLeft--
		return TransientPublicationError{Err: errors.New("stream full")}
	}
	o.published++
	return nil
}
