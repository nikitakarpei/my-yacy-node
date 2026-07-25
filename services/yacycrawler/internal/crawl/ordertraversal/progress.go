package ordertraversal

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
)

type TraversalProgress interface {
	PageDisposed(reason disposal.Reason)
	RefusalHonored(demand refusal.Demand)
	BudgetExhausted()
}
