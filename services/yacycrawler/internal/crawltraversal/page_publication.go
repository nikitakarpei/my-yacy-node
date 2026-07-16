package crawltraversal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

const (
	msgPublicationBackpressure = "publication backpressure, awaiting retry"
	msgFormatUnrepresentable   = "page disposed: no representation accepts its content format"
)

func (c *crawl) publish(ctx context.Context, page crawlcapability.CrawledPage) error {
	contents, err := c.renderContents(page)
	if err != nil {
		return err
	}
	published := 0
	for _, feed := range c.feeds {
		content, rendered := contents[feed.ContentFormat()]
		if !rendered {
			continue
		}
		publish, err := feed.Derive(page, content)
		if err != nil {
			return fmt.Errorf("derive %s: %w", feed.Representation(), err)
		}
		if err := c.send(ctx, page, feed, publish); err != nil {
			return err
		}
		c.observer.PagePublished(string(feed.Representation()))
		published++
	}
	if published == 0 {
		slog.WarnContext(ctx, msgFormatUnrepresentable,
			slog.String("url", page.CanonicalURL),
			slog.String("format", string(page.Format)),
		)
		c.observer.PageDisposed(crawlcapability.DisposalUnrepresentable)
	}
	return nil
}

func (c *crawl) renderContents(
	page crawlcapability.CrawledPage,
) (map[crawlcapability.PageContentFormat][]byte, error) {
	contents := make(map[crawlcapability.PageContentFormat][]byte, len(c.renderings))
	for _, rendering := range c.renderings {
		if !slices.Contains(rendering.SourceFormats(), page.Format) {
			continue
		}
		content, err := rendering.Render(page.Body, page.Format)
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", rendering.Format(), err)
		}
		contents[rendering.Format()] = content
	}
	return contents, nil
}

func (c *crawl) send(
	ctx context.Context,
	page crawlcapability.CrawledPage,
	feed crawlcapability.PageFeed,
	publish crawlcapability.PublishPage,
) error {
	policy := c.newBackoff(ctx)
	waits := 0
	for {
		err := publish(ctx)
		if err == nil {
			return nil
		}
		var retryable crawlcapability.TransientPublicationError
		if !errors.As(err, &retryable) {
			return fmt.Errorf("publish to %s: %w", feed.Representation(), err)
		}
		c.observer.PublicationWaited()
		waits++
		slog.WarnContext(ctx, msgPublicationBackpressure,
			slog.String("feed", string(feed.Representation())),
			slog.String("url", page.CanonicalURL),
			slog.Int("waits", waits),
			slog.Any("error", err),
		)
		if sleepErr := c.clock.Sleep(ctx, policy.NextBackOff()); sleepErr != nil {
			return fmt.Errorf("await publication retry: %w", sleepErr)
		}
	}
}
