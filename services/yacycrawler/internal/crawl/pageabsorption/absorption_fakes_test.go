package pageabsorption_test

import (
	"context"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagepublication"
)

type fakeExtract struct {
	document contentextraction.ExtractedDocument
	err      error
}

func (f fakeExtract) DocumentFrom(
	_ context.Context,
	_, _ string,
	_ []byte,
) (contentextraction.ExtractedDocument, error) {
	return f.document, f.err
}

func document(title, text string) contentextraction.ExtractedDocument {
	return contentextraction.ExtractedDocument{
		Title:  title,
		Body:   []byte(text),
		Format: contentformatgraph.FormatReadableText,
	}
}

func refusingDocument() contentextraction.ExtractedDocument {
	return contentextraction.ExtractedDocument{
		Body:            []byte("b"),
		Format:          contentformatgraph.FormatReadableText,
		RefusesIndexing: true,
	}
}

func linkingDocument(discovered string) contentextraction.ExtractedDocument {
	return contentextraction.ExtractedDocument{
		Body:           []byte("b"),
		Format:         contentformatgraph.FormatReadableText,
		DiscoveredURLs: []string{discovered},
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

type manualClock struct{ now time.Time }

func (c *manualClock) Now() time.Time { return c.now }

func (c *manualClock) Sleep(context.Context, time.Duration) error { return nil }

func newAbsorber(
	extractor pageabsorption.PageExtractor,
	publisher pageabsorption.PagePublisher,
) pageabsorption.Absorber {
	return pageabsorption.New(extractor, publisher, &manualClock{}).
		AbsorberFor(pageabsorption.Honored)
}

func succeeded(finalURL string) pagefetch.FetchedPage {
	return pagefetch.FetchedPage{
		FinalURL:    finalURL,
		ContentType: "text/html",
		Body:        []byte("x"),
	}
}
