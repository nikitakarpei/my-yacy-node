package pageabsorption

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

type fakeExtract struct {
	documents []contentextraction.ExtractedDocument
	err       error
}

func (f fakeExtract) Extract(
	_ context.Context,
	_, _ string,
	_ []byte,
) ([]contentextraction.ExtractedDocument, error) {
	return f.documents, f.err
}

func document(url, title, text string) contentextraction.ExtractedDocument {
	return contentextraction.ExtractedDocument{
		URL: url,
		ExtractedContent: contentextraction.ExtractedContent{
			Title:  title,
			Body:   []byte(text),
			Format: contentformatgraph.FormatReadableText,
		},
	}
}

type redirectEdge struct{ requested, canonical string }

type recordingResolve struct {
	mu    sync.Mutex
	edges []redirectEdge
}

func (r *recordingResolve) Record(_ context.Context, requested, canonical string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.edges = append(r.edges, redirectEdge{requested: requested, canonical: canonical})
	return nil
}

func (r *recordingResolve) recorded() []redirectEdge {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]redirectEdge(nil), r.edges...)
}

type fakeFeed struct {
	representation yacycrawlcontract.PageRepresentationKind
	contentFormat  contentformatgraph.Format
	mu             sync.Mutex
	published      []string
	failWith       error
}

func (o *fakeFeed) Representation() yacycrawlcontract.PageRepresentationKind {
	return o.representation
}

func (o *fakeFeed) ContentFormat() contentformatgraph.Format {
	if o.contentFormat == "" {
		return contentformatgraph.FormatReadableText
	}
	return o.contentFormat
}

func (o *fakeFeed) Frame(
	page CrawledPage,
	_ []byte,
) (PagePublication, error) {
	return NewPagePublication([]byte(page.CanonicalURL)), nil
}

func (o *fakeFeed) Publish(_ context.Context, publication PagePublication) error {
	if o.failWith != nil {
		return o.failWith
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.published = append(o.published, string(publication.Messages()[0]))
	return nil
}

func feeds(items ...*fakeFeed) []Feed {
	bound := make([]Feed, len(items))
	for i, item := range items {
		bound[i] = item
	}
	return bound
}

type fakeDerivation struct {
	sourceFormat contentformatgraph.Format
	targetFormat contentformatgraph.Format
}

func (d fakeDerivation) SourceFormat() contentformatgraph.Format { return d.sourceFormat }
func (d fakeDerivation) TargetFormat() contentformatgraph.Format { return d.targetFormat }

func (fakeDerivation) Derive(_ string, body []byte) ([]byte, error) { return body, nil }

func derivations() []contentformatgraph.Derivation {
	return []contentformatgraph.Derivation{
		fakeDerivation{
			sourceFormat: contentformatgraph.FormatDocumentHTML,
			targetFormat: contentformatgraph.FormatReadableText,
		},
		fakeDerivation{
			sourceFormat: contentformatgraph.FormatReadableText,
			targetFormat: contentformatgraph.FormatReadableText,
		},
		fakeDerivation{
			sourceFormat: contentformatgraph.FormatDocumentHTML,
			targetFormat: contentformatgraph.FormatMarkdown,
		},
	}
}

type recordingObserver struct {
	mu        sync.Mutex
	disposed  map[string]int
	published map[string]int
	waits     int
}

func newObserver() *recordingObserver {
	return &recordingObserver{
		disposed:  map[string]int{},
		published: map[string]int{},
	}
}

func (o *recordingObserver) PageDisposed(reason string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.disposed[reason]++
}

func (o *recordingObserver) PagePublished(out string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.published[out]++
}

func (o *recordingObserver) PublicationWaited() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.waits++
}

type manualClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *manualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *manualClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("manual clock: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	return nil
}

func newAbsorption(
	extract DocumentExtractor,
	resolve RedirectResolver,
	feeds []Feed,
	observer Progress,
) *Absorption {
	return New(
		contentformatgraph.New(derivations()),
		extract,
		resolve,
		feeds,
		observer,
		&manualClock{},
		Config{PublishRetryFloor: time.Millisecond, PublishRetryCeiling: time.Millisecond},
	)
}

func succeeded(finalURL string) pagevisit.FetchOutcome {
	return pagevisit.FetchOutcome{
		Status:      pagevisit.FetchSucceeded,
		FinalURL:    finalURL,
		ContentType: "text/html",
		Body:        []byte("x"),
	}
}
