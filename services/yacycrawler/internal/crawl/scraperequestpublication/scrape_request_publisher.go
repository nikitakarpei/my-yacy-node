// Package scraperequestpublication records a page the crawl fetched and published, for consumers to scrape.
package scraperequestpublication

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const msgScrapeRequestURLRejected = "scrape request url rejected, publication skipped"

type PublicationProgress interface {
	ScrapeRequestPublished()
}

type ScrapeRequests interface {
	Publish(ctx context.Context, canonicalURL canonicalurl.CanonicalURL) error
}

type Publisher struct {
	observer       PublicationProgress
	scrapeRequests ScrapeRequests
}

func NewPublisher(observer PublicationProgress, scrapeRequests ScrapeRequests) *Publisher {
	return &Publisher{observer: observer, scrapeRequests: scrapeRequests}
}

func (p *Publisher) Publish(ctx context.Context, finalURL string) error {
	canonicalURL, err := canonicalurl.CanonicalURLOf(finalURL)
	if err != nil {
		slog.WarnContext(ctx, msgScrapeRequestURLRejected,
			slog.String("url", finalURL),
			slog.Any("error", err),
		)
		return nil
	}
	if err := p.scrapeRequests.Publish(ctx, canonicalURL); err != nil {
		return fmt.Errorf("publish scrape request %s: %w", canonicalURL, err)
	}
	p.observer.ScrapeRequestPublished()
	return nil
}
