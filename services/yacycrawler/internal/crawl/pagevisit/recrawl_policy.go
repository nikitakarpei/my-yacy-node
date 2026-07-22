package pagevisit

import "context"

type RecrawlPolicy interface {
	Due(ctx context.Context, canonicalURL string) (bool, error)
}
