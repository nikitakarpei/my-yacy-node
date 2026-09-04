package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type CrawledPages interface {
	ReportIndexablePage(ctx context.Context, pageURL canonicalurl.CanonicalURL)
	ReportIndexingRefusedPage(ctx context.Context, pageURL canonicalurl.CanonicalURL)
}
