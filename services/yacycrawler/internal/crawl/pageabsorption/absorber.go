// Package pageabsorption turns a fetched page into its disposal outcome and discovered links.
package pageabsorption

import (
	"context"
	"errors"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
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

type absorber struct {
	extractor       PageExtractor
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
