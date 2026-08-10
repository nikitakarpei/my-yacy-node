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

type Absorber interface {
	Absorb(ctx context.Context, page fetchedpage.Page) (AbsorptionOutcome, error)
}

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

type absorber struct {
	extractor       PageExtractor
	publisher       PagePublisher
	clock           clock.Clock
	indexingRefusal IndexingRefusal
}

func (a *absorber) Absorb(
	ctx context.Context,
	page fetchedpage.Page,
) (AbsorptionOutcome, error) {
	if page.Truncated {
		return AbsorptionOutcome{Disposal: disposal.Oversized}, nil
	}
	return a.absorbDocuments(ctx, page)
}

func (a *absorber) absorbDocuments(
	ctx context.Context,
	page fetchedpage.Page,
) (AbsorptionOutcome, error) {
	documents, err := a.extractor.ExtractDocuments(
		ctx, page.FinalURL, page.ContentType, page.Body,
	)
	if err != nil {
		slog.WarnContext(ctx, msgExtractionFailed,
			slog.String("url", page.FinalURL),
			slog.Any("error", err),
		)
		return AbsorptionOutcome{Disposal: extractionDisposal(err)}, nil
	}
	if len(documents) == 0 {
		return AbsorptionOutcome{Disposal: disposal.Unextractable}, nil
	}

	var absorbed []absorbedDocument
	for _, document := range documents {
		documentAbsorption, err := a.absorbDocument(ctx, page, document)
		if err != nil {
			return AbsorptionOutcome{}, err
		}
		absorbed = append(absorbed, documentAbsorption)
	}
	return absorptionOutcomeFrom(absorbed), nil
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

func (a *absorber) absorbDocument(
	ctx context.Context,
	page fetchedpage.Page,
	document contentextraction.ExtractedDocument,
) (absorbedDocument, error) {
	canonical, err := canonicalurl.Canonicalize(document.URL)
	if err != nil {
		slog.WarnContext(ctx, msgDocumentURLRejected,
			slog.String("url", document.URL),
			slog.Any("error", err),
		)
		return absorbedDocument{disposal: disposal.UncanonicalizableURL}, nil
	}

	absorbed := absorbedDocument{discoveredURLs: a.discoverLinks(page, document)}
	if a.indexingRefusal == Honored && (document.RefusesIndexing || page.RefusesIndexing) {
		absorbed.disposal = disposal.IndexingRefused
		return absorbed, nil
	}
	if err := a.publishDocument(ctx, canonical, document); err != nil {
		return absorbedDocument{}, err
	}
	return absorbed, nil
}

func (a *absorber) discoverLinks(
	page fetchedpage.Page,
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
