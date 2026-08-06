package pagevisit

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
)

type VisitProgress interface {
	PageFetched()
	RefusalHonored(demand refusal.Demand)
	FetchCompleted(elapsed time.Duration)
}
