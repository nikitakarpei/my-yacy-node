package pageabsorption

import (
	"context"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchedpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagepublication"
)

type fakeExtract struct {
	documents []contentextraction.ExtractedDocument
	err       error
}

func (f fakeExtract) ExtractDocuments(
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

type recordingPublisher struct {
	mu       sync.Mutex
	pages    []pagepublication.Page
	failWith error
}

func (p *recordingPublisher) Publish(_ context.Context, page pagepublication.Page) error {
	if p.failWith != nil {
		return p.failWith
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pages = append(p.pages, page)
	return nil
}

func (p *recordingPublisher) published() []pagepublication.Page {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]pagepublication.Page(nil), p.pages...)
}

type recordingObserver struct {
	mu       sync.Mutex
	disposed map[disposal.Reason]int
}

func newObserver() *recordingObserver {
	return &recordingObserver{disposed: map[disposal.Reason]int{}}
}

func (o *recordingObserver) PageDisposed(reason disposal.Reason) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.disposed[reason]++
}

type manualClock struct{ now time.Time }

func (c *manualClock) Now() time.Time { return c.now }

func (c *manualClock) Sleep(context.Context, time.Duration) error { return nil }

func newAbsorber(
	extractor PageExtractor,
	publisher PagePublisher,
	observer AbsorptionProgress,
) *Absorber {
	return New(extractor, publisher, observer, &manualClock{})
}

func succeeded(finalURL string) fetchedpage.Page {
	return fetchedpage.Page{
		FinalURL:    finalURL,
		ContentType: "text/html",
		Body:        []byte("x"),
	}
}
