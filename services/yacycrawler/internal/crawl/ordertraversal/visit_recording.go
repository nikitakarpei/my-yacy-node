package ordertraversal

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

const (
	msgDeferralsExhausted = "url dropped after exhausting deferrals"
	msgFetchAbandoned     = "fetch abandoned after retries"
)

func (c *orderTraversal) recordVisit(
	ctx context.Context,
	r pageVisitResult,
	cancel context.CancelFunc,
) {
	c.inflight--
	if r.err != nil {
		if c.fatal == nil {
			c.fatal = r.err
		}
		cancel()
		return
	}
	switch r.outcome.Classification {
	case pagevisit.Deferred:
		c.deferEntry(ctx, r.entry, r.outcome.DeferFor)
	case pagevisit.Transient:
		c.recordTransient(ctx, r.entry)
	case pagevisit.NotAPage:
		c.counted++
	case pagevisit.Succeeded:
		c.counted++
		for _, link := range discoveredLinks(r.outcome.DiscoveredURLs, r.entry.Depth+1) {
			c.frontier.Admit(link.url, link.depth)
		}
	}
}

func (c *orderTraversal) recordTransient(ctx context.Context, e entry) {
	if e.Attempts >= c.config.FetchRetryLimit {
		c.observer.PageFetched()
		c.observer.PageDisposed(pagevisit.DisposalFetchFailed)
		slog.WarnContext(ctx, msgFetchAbandoned, slog.String("url", e.URL))
		c.counted++
		return
	}
	c.retryEntry(e)
}

func (c *orderTraversal) deferEntry(
	ctx context.Context,
	e entry,
	deferFor time.Duration,
) {
	if e.Deferrals >= c.config.MaxDeferralsPerURL {
		slog.WarnContext(ctx, msgDeferralsExhausted, slog.String("url", e.URL))
		c.observer.PageDisposed(pagevisit.DisposalFetchFailed)
		return
	}
	c.observer.RefusalHonored(RefusalDefer)
	e.Deferrals++
	e.NotBefore = c.clock.Now().Add(deferFor)
	c.frontier.Defer(e)
}

func (c *orderTraversal) retryEntry(e entry) {
	e.Attempts++
	e.NotBefore = c.clock.Now().Add(c.retryDelay(e.Attempts))
	c.frontier.Defer(e)
}

func (c *orderTraversal) retryDelay(attempt int) time.Duration {
	delay := c.config.FetchRetryFloor << (attempt - 1)
	if delay <= 0 || delay > c.config.FetchRetryCeiling {
		return c.config.FetchRetryCeiling
	}
	return delay
}
