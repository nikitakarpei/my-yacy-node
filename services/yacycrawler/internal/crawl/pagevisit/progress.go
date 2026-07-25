package pagevisit

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
)

type VisitProgress interface {
	PageFetched()
	PageDisposed(reason disposal.Reason)
	RefusalHonored(demand refusal.Demand)
	FetchCompleted(elapsed time.Duration)
}
