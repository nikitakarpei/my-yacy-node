// Package pageabsorption turns a fetched page into a published document and discovered links.
package pageabsorption

import (
	"context"
	"errors"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/clock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagepublication"
)

const (
	msgPageURLRejected  = "fetched page url rejected"
	msgExtractionFailed = "document extraction failed"
)

type Absorber interface {
	Absorb(ctx context.Context, page pagefetch.FetchedPage) (AbsorptionOutcome, error)
}

type PageExtractor interface {
	DocumentFrom(
		ctx context.Context,
		pageURL, contentType string,
		body []byte,
	) (contentextraction.ExtractedDocument, error)
}

type PagePublisher interface {
	Publish(ctx context.Context, page pagepublication.Page) error
}

type absorber struct {
	extractor       PageExtractor
	publisher       PagePublisher
	clock           clock.Clock
	indexingRefusal IndexingRefusal
}

func (a *absorber) Absorb(
	ctx context.Context,
	page pagefetch.FetchedPage,
) (AbsorptionOutcome, error) {
	if page.Truncated {
		return AbsorptionOutcome{Disposal: disposal.Oversized}, nil
	}
	canonical, err := canonicalurl.Canonicalize(page.FinalURL)
	if err != nil {
		slog.WarnContext(ctx, msgPageURLRejected,
			slog.String("url", page.FinalURL),
			slog.Any("error", err),
		)
		return AbsorptionOutcome{Disposal: disposal.UncanonicalizableURL}, nil
	}
	document, err := a.extractor.DocumentFrom(ctx, canonical, page.ContentType, page.Body)
	if err != nil {
		slog.WarnContext(ctx, msgExtractionFailed,
			slog.String("url", canonical),
			slog.Any("error", err),
		)
		return AbsorptionOutcome{Disposal: extractionDisposal(err)}, nil
	}

	outcome := AbsorptionOutcome{DiscoveredURLs: discoveredLinksOf(page, document)}
	if a.indexingRefusal == Honored && (document.RefusesIndexing || page.RefusesIndexing) {
		outcome.Disposal = disposal.IndexingRefused
		return outcome, nil
	}
	if err := a.publishDocument(ctx, canonical, document); err != nil {
		return AbsorptionOutcome{}, err
	}
	return outcome, nil
}

func extractionDisposal(err error) disposal.Reason {
	if errors.Is(err, contentextraction.ErrUnsupportedMediaType) {
		return disposal.UnsupportedMediaType
	}
	return disposal.Unextractable
}

func discoveredLinksOf(
	page pagefetch.FetchedPage,
	document contentextraction.ExtractedDocument,
) []string {
	if document.RefusesLinkDiscovery || page.RefusesLinkDiscovery {
		return nil
	}
	return document.DiscoveredURLs
}

func (a *absorber) publishDocument(
	ctx context.Context,
	canonical string,
	document contentextraction.ExtractedDocument,
) error {
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
