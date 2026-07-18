package crawltraversal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlfrontier"
)

const (
	msgFetchAbandoned       = "fetch abandoned after retries"
	msgDocumentURLRejected  = "extracted document url rejected"
	msgExtractionFailed     = "document extraction failed"
	msgRedirectURLRejected  = "redirect chain url rejected"
	msgRedirectRecordFailed = "redirect resolution record failed"
)

func (c *crawl) visit(
	ctx context.Context,
	entry crawlfrontier.Entry,
) visitOutcome {
	due, err := c.recrawl.Due(ctx, entry.URL)
	if err != nil {
		return visitOutcome{entry: entry, err: fmt.Errorf("recrawl decision: %w", err)}
	}
	if !due {
		return visitOutcome{entry: entry}
	}

	outcome, err := c.fetchPage(ctx, entry.URL)
	if err != nil {
		return visitOutcome{entry: entry, err: err}
	}

	switch outcome.Status {
	case crawlcapability.FetchCeased:
		c.observer.RefusalHonored(crawlcapability.RefusalCease)
		c.observer.PageDisposed(crawlcapability.DisposalRefused)
		return visitOutcome{entry: entry}
	case crawlcapability.FetchDeferred:
		return visitOutcome{entry: entry, deferred: true, deferFor: outcome.DeferFor}
	case crawlcapability.FetchNotAPage:
		c.observer.PageFetched()
		c.observer.PageDisposed(crawlcapability.DisposalFetchFailed)
		return visitOutcome{entry: entry, counted: true}
	case crawlcapability.FetchTransient:
		if entry.Attempts >= c.config.FetchRetryLimit {
			c.observer.PageFetched()
			c.observer.PageDisposed(crawlcapability.DisposalFetchFailed)
			slog.WarnContext(ctx, msgFetchAbandoned, slog.String("url", entry.URL))
			return visitOutcome{entry: entry, counted: true}
		}
		return visitOutcome{entry: entry, transient: true}
	}

	c.observer.PageFetched()
	c.recordRedirects(ctx, outcome)
	if outcome.Truncated {
		c.observer.PageDisposed(crawlcapability.DisposalOversized)
		return visitOutcome{entry: entry, counted: true}
	}
	return c.absorbPage(ctx, entry, outcome)
}

func (c *crawl) recordRedirects(ctx context.Context, outcome crawlcapability.FetchOutcome) {
	canonicalFinal, err := canonicalurl.Canonicalize(outcome.FinalURL)
	if err != nil {
		slog.WarnContext(ctx, msgRedirectURLRejected,
			slog.String("url", outcome.FinalURL),
			slog.Any("error", err),
		)
		return
	}
	for _, hop := range outcome.RedirectChain {
		canonicalHop, err := canonicalurl.Canonicalize(hop)
		if err != nil {
			slog.WarnContext(ctx, msgRedirectURLRejected,
				slog.String("url", hop),
				slog.Any("error", err),
			)
			continue
		}
		if canonicalHop == canonicalFinal {
			continue
		}
		if err := c.resolve.Record(ctx, canonicalHop, canonicalFinal); err != nil {
			slog.WarnContext(ctx, msgRedirectRecordFailed,
				slog.String("requested", canonicalHop),
				slog.String("canonical", canonicalFinal),
				slog.Any("error", err),
			)
		}
	}
}

func (c *crawl) absorbPage(
	ctx context.Context,
	entry crawlfrontier.Entry,
	outcome crawlcapability.FetchOutcome,
) visitOutcome {
	documents, err := c.extract.Extract(ctx, outcome.FinalURL, outcome.ContentType, outcome.Body)
	if err != nil {
		slog.WarnContext(ctx, msgExtractionFailed,
			slog.String("url", entry.URL),
			slog.Any("error", err),
		)
		switch {
		case errors.Is(err, crawlcapability.ErrUnsupportedMediaType):
			c.observer.PageDisposed(crawlcapability.DisposalUnsupportedMediaType)
		case errors.Is(err, crawlcapability.ErrContainerOverflow):
			c.observer.PageDisposed(crawlcapability.DisposalContainerOverflow)
		default:
			c.observer.PageDisposed(crawlcapability.DisposalUnextractable)
		}
		return visitOutcome{entry: entry, counted: true}
	}
	if len(documents) == 0 {
		c.observer.PageDisposed(crawlcapability.DisposalUnextractable)
		return visitOutcome{entry: entry, counted: true}
	}

	result := visitOutcome{entry: entry, counted: true}
	for _, document := range documents {
		candidates, err := c.absorbDocument(ctx, entry, outcome, document)
		if err != nil {
			return visitOutcome{entry: entry, err: err}
		}
		result.candidates = append(result.candidates, candidates...)
	}
	return result
}

func (c *crawl) absorbDocument(
	ctx context.Context,
	entry crawlfrontier.Entry,
	outcome crawlcapability.FetchOutcome,
	document crawlcapability.ExtractedDocument,
) ([]discoveredLink, error) {
	canonical, err := canonicalurl.Canonicalize(document.URL)
	if err != nil {
		slog.WarnContext(ctx, msgDocumentURLRejected,
			slog.String("url", document.URL),
			slog.Any("error", err),
		)
		c.observer.PageDisposed(crawlcapability.DisposalUnextractable)
		return nil, nil
	}

	candidates := c.discoverLinks(entry, outcome, document)
	if err := c.publishDocument(ctx, canonical, document, outcome); err != nil {
		return nil, err
	}
	return candidates, nil
}

func (c *crawl) discoverLinks(
	entry crawlfrontier.Entry,
	outcome crawlcapability.FetchOutcome,
	document crawlcapability.ExtractedDocument,
) []discoveredLink {
	if document.RefusesLinkDiscovery || outcome.RefusesLinkDiscovery {
		return nil
	}
	links := make([]discoveredLink, 0, len(document.Links))
	for _, link := range document.Links {
		links = append(links, discoveredLink{url: link, depth: entry.Depth + 1})
	}
	return links
}

func (c *crawl) publishDocument(
	ctx context.Context,
	canonical string,
	document crawlcapability.ExtractedDocument,
	outcome crawlcapability.FetchOutcome,
) error {
	if document.RefusesIndexing || outcome.RefusesIndexing {
		c.observer.PageDisposed(crawlcapability.DisposalNoIndex)
		return nil
	}
	page := crawlcapability.CrawledPage{
		CanonicalURL:      canonical,
		Title:             document.Title,
		Body:              document.Body,
		Format:            document.Format,
		Language:          document.Language,
		CrawledAt:         c.clock.Now(),
		LocalLinkCount:    document.LocalLinkCount,
		ExternalLinkCount: document.ExternalLinkCount,
	}
	return c.publish(ctx, page)
}
