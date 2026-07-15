package crawltraversal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

const (
	msgPublicationBackpressure = "publication backpressure, awaiting retry"
	msgFormatUnrepresentable   = "page disposed: no representation accepts its content format"
)

func (c *crawl) publish(ctx context.Context, page crawlcapability.CrawledPage) error {
	accepting := c.accepting(page.Format)
	if len(accepting) == 0 {
		slog.WarnContext(ctx, msgFormatUnrepresentable,
			slog.String("url", page.CanonicalURL),
			slog.String("format", string(page.Format)),
		)
		c.observer.PageDisposed(crawlcapability.DisposalUnrepresentable)
		return nil
	}
	rendered := crawlcapability.NewRenderedContent(page.Body, page.Format)
	for _, output := range accepting {
		send, err := output.Prepare(page, rendered)
		if err != nil {
			return fmt.Errorf("prepare %s: %w", output.Name(), err)
		}
		if err := c.send(ctx, page, output, send); err != nil {
			return err
		}
		c.observer.PagePublished(output.Name())
	}
	return nil
}

func (c *crawl) send(
	ctx context.Context,
	page crawlcapability.CrawledPage,
	output crawlcapability.PageRepresentationOutput,
	send func(context.Context) error,
) error {
	policy := c.newBackoff(ctx)
	waits := 0
	for {
		err := send(ctx)
		if err == nil {
			return nil
		}
		var retryable crawlcapability.TransientPublicationError
		if !errors.As(err, &retryable) {
			return fmt.Errorf("publish to %s: %w", output.Name(), err)
		}
		c.observer.PublicationWaited()
		waits++
		slog.WarnContext(ctx, msgPublicationBackpressure,
			slog.String("output", output.Name()),
			slog.String("url", page.CanonicalURL),
			slog.Int("waits", waits),
			slog.Any("error", err),
		)
		if sleepErr := c.clock.Sleep(ctx, policy.NextBackOff()); sleepErr != nil {
			return fmt.Errorf("await publication retry: %w", sleepErr)
		}
	}
}

func (c *crawl) accepting(
	format crawlcapability.PageContentFormat,
) []crawlcapability.PageRepresentationOutput {
	accepting := make([]crawlcapability.PageRepresentationOutput, 0, len(c.outputs))
	for _, output := range c.outputs {
		if output.Accepts(format) {
			accepting = append(accepting, output)
		}
	}
	return accepting
}
