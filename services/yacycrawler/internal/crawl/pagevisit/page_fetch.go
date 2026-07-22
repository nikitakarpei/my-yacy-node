package pagevisit

import (
	"context"
	"fmt"
)

func (v *PageVisit) fetchPage(
	ctx context.Context,
	rawURL string,
) (FetchOutcome, error) {
	start := v.clock.Now()
	outcome, err := v.fetch.Fetch(ctx, rawURL)
	if err != nil {
		return FetchOutcome{}, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	v.observer.FetchObserved(v.clock.Now().Sub(start))
	return outcome, nil
}
