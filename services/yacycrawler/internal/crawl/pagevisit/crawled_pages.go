package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type CrawledPages interface {
	PublishIndexablePage(ctx context.Context, page yacycrawlcontract.CrawledPage) bool
	PublishIndexingRefusedPage(ctx context.Context, page yacycrawlcontract.CrawledPage) bool
}
