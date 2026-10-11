// Package redis keeps every crawl order the crawler accepted for the retention of
// accepted orders, so that any worker can rebuild the profile a URL of that order was
// admitted under.
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/redisrecord"
)

const recordKind = "YACY_ACCEPTED_ORDERS"

type Orders struct {
	client    redis.UniversalClient
	retention time.Duration
}

func New(client redis.UniversalClient, retention time.Duration) *Orders {
	return &Orders{client: client, retention: retention}
}

func (o *Orders) Keep(ctx context.Context, order acceptedorder.AcceptedOrder) error {
	data, err := yacycrawlcontract.MarshalCrawlOrder(order.CrawlOrder())
	if err != nil {
		return fmt.Errorf("keep order %s: %w", order.OrderID(), err)
	}
	if err := o.client.Set(ctx, orderKeyOf(order.OrderID()), data, o.retention).Err(); err != nil {
		return fmt.Errorf("keep order %s: %w", order.OrderID(), err)
	}
	return nil
}

func (o *Orders) OrderOf(
	ctx context.Context,
	orderID string,
) (acceptedorder.AcceptedOrder, error) {
	data, err := o.client.Get(ctx, orderKeyOf(orderID)).Bytes()
	if err != nil {
		return acceptedorder.AcceptedOrder{}, fmt.Errorf("read order %s: %w", orderID, err)
	}
	storedOrder, err := yacycrawlcontract.UnmarshalCrawlOrder(data)
	if err != nil {
		return acceptedorder.AcceptedOrder{}, err
	}
	return acceptedorder.AcceptedOrderFrom(storedOrder)
}

func orderKeyOf(orderID string) string {
	return redisrecord.KeyOf(recordKind, orderID)
}
