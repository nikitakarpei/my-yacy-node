package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type CrawledPages interface {
	PublishIndexablePage(ctx context.Context, pageURL canonicalurl.CanonicalURL)
	PublishIndexingRefusedPage(ctx context.Context, pageURL canonicalurl.CanonicalURL)
}
