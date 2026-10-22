package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

type LinkAdmissionObserver interface {
	LinksUnadmitted(ctx context.Context, crawlID crawlid.CrawlID, amountOfLinks int)
}
