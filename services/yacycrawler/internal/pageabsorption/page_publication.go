package pageabsorption

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/cenkalti/backoff/v4"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

const (
	msgPublicationBackpressure = "publication backpressure, awaiting retry"
	msgFormatUnrepresentable   = "page disposed: no representation accepts its content format"
)

func (a *Absorption) publish(ctx context.Context, page crawlcapability.CrawledPage) error {
	contents, err := a.deriveContents(page)
	if err != nil {
		return err
	}
	published := 0
	for _, feed := range a.feeds {
		content, framed := contents[feed.ContentFormat()]
		if !framed {
			continue
		}
		publication, err := feed.Frame(page, content)
		if err != nil {
			return fmt.Errorf("frame %s: %w", feed.Representation(), err)
		}
		if err := a.send(ctx, page, feed, publication); err != nil {
			return err
		}
		a.observer.PagePublished(string(feed.Representation()))
		published++
	}
	if published == 0 {
		slog.WarnContext(ctx, msgFormatUnrepresentable,
			slog.String("url", page.CanonicalURL),
			slog.String("format", string(page.Format)),
		)
		a.observer.PageDisposed(crawlcapability.DisposalUnrepresentable)
	}
	return nil
}

func (a *Absorption) deriveContents(
	page crawlcapability.CrawledPage,
) (map[crawlcapability.PageContentFormat][]byte, error) {
	resolver := a.graph.Resolver(page.CanonicalURL, page.Format, page.Body)
	for _, feed := range a.feeds {
		if _, _, err := resolver.Resolve(feed.ContentFormat()); err != nil {
			return nil, err
		}
	}
	return resolver.Contents(), nil
}

func (a *Absorption) send(
	ctx context.Context,
	page crawlcapability.CrawledPage,
	feed crawlcapability.PageFeed,
	publication crawlcapability.PagePublication,
) error {
	policy := a.newBackoff(ctx)
	waits := 0
	for {
		err := feed.Publish(ctx, publication)
		if err == nil {
			return nil
		}
		var retryable crawlcapability.TransientPublicationError
		if !errors.As(err, &retryable) {
			return fmt.Errorf("publish to %s: %w", feed.Representation(), err)
		}
		a.observer.PublicationWaited()
		waits++
		slog.WarnContext(ctx, msgPublicationBackpressure,
			slog.String("feed", string(feed.Representation())),
			slog.String("url", page.CanonicalURL),
			slog.Int("waits", waits),
			slog.Any("error", err),
		)
		if sleepErr := a.clock.Sleep(ctx, policy.NextBackOff()); sleepErr != nil {
			return fmt.Errorf("await publication retry: %w", sleepErr)
		}
	}
}

func (a *Absorption) newBackoff(ctx context.Context) backoff.BackOff {
	exponential := backoff.NewExponentialBackOff()
	exponential.InitialInterval = a.config.PublishRetryFloor
	exponential.MaxInterval = a.config.PublishRetryCeiling
	exponential.MaxElapsedTime = 0
	return backoff.WithContext(exponential, ctx)
}
