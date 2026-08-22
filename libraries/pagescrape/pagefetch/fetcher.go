// Package pagefetch is the contract between a page fetcher and its caller.
package pagefetch

import "context"

type Fetcher interface {
	Fetch(ctx context.Context, rawURL string, knownVersion PageVersion) (FetchOutcome, error)
}
