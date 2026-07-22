package pagevisit

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

func (v *PageVisit) fetchPage(
	ctx context.Context,
	rawURL string,
) (crawlcapability.FetchOutcome, error) {
	start := v.clock.Now()
	outcome, err := v.fetch.Fetch(ctx, rawURL)
	if err != nil {
		return crawlcapability.FetchOutcome{}, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	v.observer.FetchObserved(v.clock.Now().Sub(start))
	return outcome, nil
}
