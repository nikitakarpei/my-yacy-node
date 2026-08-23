package ordertraversal

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
)

type TraversalProgress interface {
	RefusalHonored(demand refusal.Demand)
	BudgetExhausted()
	PageDisposed(reason disposal.Reason)
}
