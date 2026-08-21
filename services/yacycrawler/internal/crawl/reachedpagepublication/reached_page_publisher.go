// Package reachedpagepublication records a page the crawl fetched and published, for consumers to scrape.
package reachedpagepublication

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const msgReachedPageURLRejected = "reached page url rejected, publication skipped"

type PublicationProgress interface {
	ReachedPagePublished()
}

type ReachedPages interface {
	Publish(ctx context.Context, canonicalURL string) error
}

type Publisher struct {
	observer PublicationProgress
	reached  ReachedPages
}

func NewPublisher(observer PublicationProgress, reached ReachedPages) *Publisher {
	return &Publisher{observer: observer, reached: reached}
}

func (p *Publisher) Publish(ctx context.Context, finalURL string) error {
	canonicalURL, err := canonicalurl.Canonicalize(finalURL)
	if err != nil {
		slog.WarnContext(ctx, msgReachedPageURLRejected,
			slog.String("url", finalURL),
			slog.Any("error", err),
		)
		return nil
	}
	if err := p.reached.Publish(ctx, canonicalURL); err != nil {
		return fmt.Errorf("publish reached page %s: %w", canonicalURL, err)
	}
	p.observer.ReachedPagePublished()
	return nil
}
