package pagevisit

import "context"

type Fetcher interface {
	Fetch(ctx context.Context, rawURL string) (FetchOutcome, error)
}
