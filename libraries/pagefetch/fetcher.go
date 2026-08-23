// Package pagefetch is the contract between a page fetcher and its caller.
package pagefetch

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type Fetcher interface {
	Fetch(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		knownVersion PageVersion,
	) (FetchOutcome, error)
}
