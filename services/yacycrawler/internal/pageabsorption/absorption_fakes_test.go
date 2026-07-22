package pageabsorption

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type fakeExtract struct {
	documents []crawlcapability.ExtractedDocument
	err       error
}

func (f fakeExtract) Extract(
	_ context.Context,
	_, _ string,
	_ []byte,
) ([]crawlcapability.ExtractedDocument, error) {
	return f.documents, f.err
}

func document(url, title, text string) crawlcapability.ExtractedDocument {
	return crawlcapability.ExtractedDocument{
		URL: url,
		ExtractedContent: crawlcapability.ExtractedContent{
			Title:  title,
			Body:   []byte(text),
			Format: crawlcapability.PageContentFormatReadableText,
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
	contentFormat  crawlcapability.PageContentFormat
	mu             sync.Mutex
	published      []string
	failWith       error
}

func (o *fakeFeed) Representation() yacycrawlcontract.PageRepresentationKind {
	return o.representation
}

func (o *fakeFeed) ContentFormat() crawlcapability.PageContentFormat {
	if o.contentFormat == "" {
		return crawlcapability.PageContentFormatReadableText
	}
	return o.contentFormat
}

func (o *fakeFeed) Frame(
	page crawlcapability.CrawledPage,
	_ []byte,
) (crawlcapability.PagePublication, error) {
	return crawlcapability.NewPagePublication([]byte(page.CanonicalURL)), nil
}

func (o *fakeFeed) Publish(_ context.Context, publication crawlcapability.PagePublication) error {
	if o.failWith != nil {
		return o.failWith
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.published = append(o.published, string(publication.Messages()[0]))
	return nil
}

func feeds(items ...*fakeFeed) []crawlcapability.PageFeed {
	bound := make([]crawlcapability.PageFeed, len(items))
	for i, item := range items {
		bound[i] = item
	}
	return bound
}

type fakeDerivation struct {
	sourceFormat crawlcapability.PageContentFormat
	targetFormat crawlcapability.PageContentFormat
}

func (d fakeDerivation) SourceFormat() crawlcapability.PageContentFormat { return d.sourceFormat }
func (d fakeDerivation) TargetFormat() crawlcapability.PageContentFormat { return d.targetFormat }

func (fakeDerivation) Derive(_ string, body []byte) ([]byte, error) { return body, nil }

func derivations() []crawlcapability.PageDerivation {
	return []crawlcapability.PageDerivation{
		fakeDerivation{
			sourceFormat: crawlcapability.PageContentFormatDocumentHTML,
			targetFormat: crawlcapability.PageContentFormatReadableText,
		},
		fakeDerivation{
			sourceFormat: crawlcapability.PageContentFormatReadableText,
			targetFormat: crawlcapability.PageContentFormatReadableText,
		},
		fakeDerivation{
			sourceFormat: crawlcapability.PageContentFormatDocumentHTML,
			targetFormat: crawlcapability.PageContentFormatMarkdown,
		},
	}
}

type recordingObserver struct {
	mu        sync.Mutex
	disposed  map[string]int
	published map[string]int
	waits     int
}

func (*recordingObserver) OrderReceived()              {}
func (*recordingObserver) OrderRedelivered()           {}
func (*recordingObserver) OrderCompleted()             {}
func (*recordingObserver) PageFetched()                {}
func (*recordingObserver) RefusalHonored(string)       {}
func (*recordingObserver) BudgetExhausted()            {}
func (*recordingObserver) FetchObserved(time.Duration) {}

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
	extract crawlcapability.DocumentExtraction,
	resolve crawlcapability.RedirectResolution,
	feeds []crawlcapability.PageFeed,
	observer crawlcapability.RunProgress,
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

func succeeded(finalURL string) crawlcapability.FetchOutcome {
	return crawlcapability.FetchOutcome{
		Status:      crawlcapability.FetchSucceeded,
		FinalURL:    finalURL,
		ContentType: "text/html",
		Body:        []byte("x"),
	}
}
