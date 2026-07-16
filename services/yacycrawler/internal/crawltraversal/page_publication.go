package crawltraversal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/renderedcontent"
)

const (
	msgPublicationBackpressure = "publication backpressure, awaiting retry"
	msgFormatUnrepresentable   = "page disposed: no representation accepts its content format"
)

func (c *crawl) publish(ctx context.Context, page crawlcapability.CrawledPage) error {
	accepting := c.feedsAccepting(page.Format)
	if len(accepting) == 0 {
		slog.WarnContext(ctx, msgFormatUnrepresentable,
			slog.String("url", page.CanonicalURL),
			slog.String("format", string(page.Format)),
		)
		c.observer.PageDisposed(crawlcapability.DisposalUnrepresentable)
		return nil
	}
	rendered := renderedcontent.New(page.Body, page.Format)
	for _, feed := range accepting {
		publish, err := feed.Derive(page, rendered.In)
		if err != nil {
			return fmt.Errorf("derive %s: %w", feed.Representation(), err)
		}
		if err := c.send(ctx, page, feed, publish); err != nil {
			return err
		}
		c.observer.PagePublished(string(feed.Representation()))
	}
	return nil
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

func (c *crawl) feedsAccepting(
	format crawlcapability.PageContentFormat,
) []crawlcapability.PageFeed {
	accepting := make([]crawlcapability.PageFeed, 0, len(c.feeds))
	for _, feed := range c.feeds {
		if feed.Accepts(format) {
			accepting = append(accepting, feed)
		}
	}
	return accepting
}
