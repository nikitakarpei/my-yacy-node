package pageabsorption

import (
	"context"
	"errors"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

const (
	msgDocumentURLRejected = "extracted document url rejected"
	msgExtractionFailed    = "document extraction failed"
)

func (a *Absorption) absorbDocuments(
	ctx context.Context,
	outcome crawlcapability.FetchOutcome,
) ([]string, error) {
	documents, err := a.extract.Extract(ctx, outcome.FinalURL, outcome.ContentType, outcome.Body)
	if err != nil {
		slog.WarnContext(ctx, msgExtractionFailed,
			slog.String("url", outcome.FinalURL),
			slog.Any("error", err),
		)
		switch {
		case errors.Is(err, crawlcapability.ErrUnsupportedMediaType):
			a.observer.PageDisposed(crawlcapability.DisposalUnsupportedMediaType)
		case errors.Is(err, crawlcapability.ErrContainerOverflow):
			a.observer.PageDisposed(crawlcapability.DisposalContainerOverflow)
		default:
			a.observer.PageDisposed(crawlcapability.DisposalUnextractable)
		}
		return nil, nil
	}
	if len(documents) == 0 {
		a.observer.PageDisposed(crawlcapability.DisposalUnextractable)
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
	outcome crawlcapability.FetchOutcome,
	document crawlcapability.ExtractedDocument,
) ([]string, error) {
	canonical, err := canonicalurl.Canonicalize(document.URL)
	if err != nil {
		slog.WarnContext(ctx, msgDocumentURLRejected,
			slog.String("url", document.URL),
			slog.Any("error", err),
		)
		a.observer.PageDisposed(crawlcapability.DisposalUnextractable)
		return nil, nil
	}

	links := a.discoverLinks(outcome, document)
	if err := a.publishDocument(ctx, canonical, document, outcome); err != nil {
		return nil, err
	}
	return links, nil
}

func (a *Absorption) discoverLinks(
	outcome crawlcapability.FetchOutcome,
	document crawlcapability.ExtractedDocument,
) []string {
	if document.RefusesLinkDiscovery || outcome.RefusesLinkDiscovery {
		return nil
	}
	return document.Links
}

func (a *Absorption) publishDocument(
	ctx context.Context,
	canonical string,
	document crawlcapability.ExtractedDocument,
	outcome crawlcapability.FetchOutcome,
) error {
	if document.RefusesIndexing || outcome.RefusesIndexing {
		a.observer.PageDisposed(crawlcapability.DisposalNoIndex)
		return nil
	}
	page := crawlcapability.CrawledPage{
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
