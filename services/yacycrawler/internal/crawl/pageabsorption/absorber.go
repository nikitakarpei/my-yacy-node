// Package pageabsorption turns a fetched page into published documents and discovered links.
package pageabsorption

import (
	"context"
	"errors"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/clock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchedpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagepublication"
)

const (
	msgDocumentURLRejected = "extracted document url rejected"
	msgExtractionFailed    = "document extraction failed"
)

type PageExtractor interface {
	ExtractDocuments(
		ctx context.Context,
		pageURL, contentType string,
		body []byte,
	) ([]contentextraction.ExtractedDocument, error)
}

type PagePublisher interface {
	Publish(ctx context.Context, page pagepublication.Page) error
}

type Absorber struct {
	extractor PageExtractor
	publisher PagePublisher
	observer  AbsorptionProgress
	clock     clock.Clock
}

func New(
	extractor PageExtractor,
	publisher PagePublisher,
	observer AbsorptionProgress,
	clock clock.Clock,
) *Absorber {
	return &Absorber{
		extractor: extractor,
		publisher: publisher,
		observer:  observer,
		clock:     clock,
	}
}

func (a *Absorber) Absorb(
	ctx context.Context,
	page fetchedpage.Page,
) ([]string, error) {
	if page.Truncated {
		a.observer.PageDisposed(disposal.Oversized)
		return nil, nil
	}
	return a.absorbDocuments(ctx, page)
}

func (a *Absorber) absorbDocuments(
	ctx context.Context,
	page fetchedpage.Page,
) ([]string, error) {
	documents, err := a.extractor.ExtractDocuments(
		ctx, page.FinalURL, page.ContentType, page.Body,
	)
	if err != nil {
		slog.WarnContext(ctx, msgExtractionFailed,
			slog.String("url", page.FinalURL),
			slog.Any("error", err),
		)
		a.observer.PageDisposed(extractionDisposal(err))
		return nil, nil
	}
	if len(documents) == 0 {
		a.observer.PageDisposed(disposal.Unextractable)
		return nil, nil
	}

	var links []string
	for _, document := range documents {
		discovered, err := a.absorbDocument(ctx, page, document)
		if err != nil {
			return nil, err
		}
		links = append(links, discovered...)
	}
	return links, nil
}

func extractionDisposal(err error) disposal.Reason {
	switch {
	case errors.Is(err, contentextraction.ErrUnsupportedMediaType):
		return disposal.UnsupportedMediaType
	case errors.Is(err, contentextraction.ErrNestingTooDeep):
		return disposal.NestingTooDeep
	case errors.Is(err, contentextraction.ErrDocumentBudgetExhausted):
		return disposal.DocumentBudgetExhausted
	default:
		return disposal.Unextractable
	}
}

func (a *Absorber) absorbDocument(
	ctx context.Context,
	page fetchedpage.Page,
	document contentextraction.ExtractedDocument,
) ([]string, error) {
	canonical, err := canonicalurl.Canonicalize(document.URL)
	if err != nil {
		slog.WarnContext(ctx, msgDocumentURLRejected,
			slog.String("url", document.URL),
			slog.Any("error", err),
		)
		a.observer.PageDisposed(disposal.UncanonicalizableURL)
		return nil, nil
	}

	links := a.discoverLinks(page, document)
	if err := a.publishDocument(ctx, canonical, document, page); err != nil {
		return nil, err
	}
	return links, nil
}

func (a *Absorber) discoverLinks(
	page fetchedpage.Page,
	document contentextraction.ExtractedDocument,
) []string {
	if document.RefusesLinkDiscovery || page.RefusesLinkDiscovery {
		return nil
	}
	return document.DiscoveredURLs
}

func (a *Absorber) publishDocument(
	ctx context.Context,
	canonical string,
	document contentextraction.ExtractedDocument,
	page fetchedpage.Page,
) error {
	if document.RefusesIndexing || page.RefusesIndexing {
		a.observer.PageDisposed(disposal.IndexingRefused)
		return nil
	}
	crawled := pagepublication.Page{
		CanonicalURL:  canonical,
		Title:         document.Title,
		Body:          document.Body,
		Format:        document.Format,
		Language:      document.Language,
		CrawledAt:     a.clock.Now(),
		LocalLinks:    document.LocalLinks,
		ExternalLinks: document.ExternalLinks,
	}
	return a.publisher.Publish(ctx, crawled)
}
