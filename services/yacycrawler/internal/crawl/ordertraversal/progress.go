package ordertraversal

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
)

type TraversalProgress interface {
	PageDisposed(reason disposal.Reason)
	RefusalHonored(demand refusal.Demand)
	BudgetExhausted()
}

type DisposedPages interface {
	Record(ctx context.Context, url string) error
}
