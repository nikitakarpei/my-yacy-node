package pageabsorption

import (
	"context"
	"errors"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

const (
	msgDocumentURLRejected = "extracted document url rejected"
	msgExtractionFailed    = "document extraction failed"
)

func (a *Absorption) absorbDocuments(
	ctx context.Context,
	outcome pagevisit.FetchOutcome,
) ([]string, error) {
	documents, err := a.extract.Extract(ctx, outcome.FinalURL, outcome.ContentType, outcome.Body)
	if err != nil {
		slog.WarnContext(ctx, msgExtractionFailed,
			slog.String("url", outcome.FinalURL),
			slog.Any("error", err),
		)
		switch {
		case errors.Is(err, contentextraction.ErrUnsupportedMediaType):
			a.observer.PageDisposed(DisposalUnsupportedMediaType)
		case errors.Is(err, contentextraction.ErrContainerOverflow):
			a.observer.PageDisposed(DisposalContainerOverflow)
		default:
			a.observer.PageDisposed(DisposalUnextractable)
		}
		return nil, nil
	}
	if len(documents) == 0 {
		a.observer.PageDisposed(DisposalUnextractable)
		return nil, nil
	}

	var links []string
	for _, document := range documents {
		discovered, err := a.absorbDocument(ctx, outcome, document)
		if err != nil {
			return nil, err
		}
		links = append(links, discovered...)
	}
	return links, nil
}

func (a *Absorption) absorbDocument(
	ctx context.Context,
	outcome pagevisit.FetchOutcome,
	document contentextraction.ExtractedDocument,
) ([]string, error) {
	canonical, err := canonicalurl.Canonicalize(document.URL)
	if err != nil {
		slog.WarnContext(ctx, msgDocumentURLRejected,
			slog.String("url", document.URL),
			slog.Any("error", err),
		)
		a.observer.PageDisposed(DisposalUnextractable)
		return nil, nil
	}

	links := a.discoverLinks(outcome, document)
	if err := a.publishDocument(ctx, canonical, document, outcome); err != nil {
		return nil, err
	}
	return links, nil
}

func (a *Absorption) discoverLinks(
	outcome pagevisit.FetchOutcome,
	document contentextraction.ExtractedDocument,
) []string {
	if document.RefusesLinkDiscovery || outcome.RefusesLinkDiscovery {
		return nil
	}
	return document.Links
}

func (a *Absorption) publishDocument(
	ctx context.Context,
	canonical string,
	document contentextraction.ExtractedDocument,
	outcome pagevisit.FetchOutcome,
) error {
	if document.RefusesIndexing || outcome.RefusesIndexing {
		a.observer.PageDisposed(DisposalNoIndex)
		return nil
	}
	page := CrawledPage{
		CanonicalURL:      canonical,
		Title:             document.Title,
		Body:              document.Body,
		Format:            document.Format,
		Language:          document.Language,
		CrawledAt:         a.clock.Now(),
		LocalLinkCount:    document.LocalLinkCount,
		ExternalLinkCount: document.ExternalLinkCount,
	}
	return a.publish(ctx, page)
}
