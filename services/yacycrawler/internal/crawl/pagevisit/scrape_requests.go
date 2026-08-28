package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type ScrapeRequests interface {
	Publish(ctx context.Context, canonicalURL canonicalurl.CanonicalURL) error
}
