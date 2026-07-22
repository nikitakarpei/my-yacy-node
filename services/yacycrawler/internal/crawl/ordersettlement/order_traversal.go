package ordersettlement

import (
	"context"
)

type OrderTraversal interface {
	Traverse(ctx context.Context, delivery DeliveredOrder) error
}
