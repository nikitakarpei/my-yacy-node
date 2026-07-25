package ordersettlement

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type OrderTraverser interface {
	Traverse(ctx context.Context, order yacycrawlcontract.CrawlOrder) error
}
