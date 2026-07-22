package crawltraversal

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlfrontier"
)

const msgDeferralsExhausted = "url dropped after exhausting deferrals"

func (c *traversal) recordVisit(ctx context.Context, r visitOutcome, cancel context.CancelFunc) {
	c.inflight--
	if r.err != nil {
		if c.fatal == nil {
			c.fatal = r.err
		}
		cancel()
		return
	}
	if r.deferred {
		c.deferEntry(ctx, r.entry, r.deferFor)
		return
	}
	if r.transient {
		c.retryEntry(r.entry)
		return
	}
	if r.counted {
		c.counted++
	}
	for _, link := range r.candidates {
		c.frontier.Admit(link.url, link.depth)
	}
}

func (c *traversal) deferEntry(
	ctx context.Context,
	entry crawlfrontier.Entry,
	deferFor time.Duration,
) {
	if entry.Deferrals >= c.config.MaxDeferralsPerURL {
		slog.WarnContext(ctx, msgDeferralsExhausted, slog.String("url", entry.URL))
		c.observer.PageDisposed(crawlcapability.DisposalFetchFailed)
		return
	}
	c.observer.RefusalHonored(crawlcapability.RefusalDefer)
	entry.Deferrals++
	entry.NotBefore = c.clock.Now().Add(deferFor)
	c.frontier.Defer(entry)
}

func (c *traversal) retryEntry(entry crawlfrontier.Entry) {
	entry.Attempts++
	entry.NotBefore = c.clock.Now().Add(c.retryDelay(entry.Attempts))
	c.frontier.Defer(entry)
}

func (c *traversal) retryDelay(attempt int) time.Duration {
	delay := c.config.FetchRetryFloor << (attempt - 1)
	if delay <= 0 || delay > c.config.FetchRetryCeiling {
		return c.config.FetchRetryCeiling
	}
	return delay
}
