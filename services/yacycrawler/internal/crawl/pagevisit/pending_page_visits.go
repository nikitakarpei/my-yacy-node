package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type PendingPageVisits interface {
	Publish(ctx context.Context, pageVisit pendingpagevisit.PendingPageVisit) error
}
