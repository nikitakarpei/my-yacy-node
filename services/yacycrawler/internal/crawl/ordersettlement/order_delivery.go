package ordersettlement

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type OrderDelivery interface {
	Order() yacycrawlcontract.CrawlOrder
	Acknowledge(ctx context.Context) error
	Return(ctx context.Context) error
	ExtendOwnership(ctx context.Context) error
}
