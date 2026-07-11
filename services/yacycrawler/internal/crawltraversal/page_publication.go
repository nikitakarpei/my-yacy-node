package crawltraversal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

const msgPublicationBackpressure = "publication backpressure, awaiting retry"

func (c *crawl) publish(ctx context.Context, page crawlcapability.ExtractedPage) error {
	for _, output := range c.outputs {
		policy := c.newBackoff(ctx)
		waits := 0
		for {
			err := output.Publish(ctx, page)
			if err == nil {
				break
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
		c.observer.PagePublished(output.Name())
	}
	return nil
}
