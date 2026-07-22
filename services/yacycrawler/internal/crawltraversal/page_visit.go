package crawltraversal

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlfrontier"
)

const msgFetchAbandoned = "fetch abandoned after retries"

func (c *traversal) visit(
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
	links, err := c.absorption.Absorb(ctx, outcome)
	if err != nil {
		return visitOutcome{entry: entry, err: err}
	}
	return visitOutcome{
		entry:      entry,
		counted:    true,
		candidates: discoveredLinks(links, entry.Depth+1),
	}
}
