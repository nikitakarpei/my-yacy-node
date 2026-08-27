// Package jetstream keeps every crawl order the crawler accepted, so that any
// worker can rebuild the profile a URL of that order was admitted under.
package jetstream

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
)

const BucketName = "YACY_ACCEPTED_ORDERS"

type BucketSpec struct {
	MaxBytes  int64
	Retention time.Duration
}

func Ensure(ctx context.Context, js jetstream.JetStream, spec BucketSpec) error {
	if _, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:   BucketName,
		MaxBytes: spec.MaxBytes,
		TTL:      spec.Retention,
	}); err != nil {
		return fmt.Errorf("ensure accepted order bucket: %w", err)
	}
	return nil
}

type Orders struct {
	bucket jetstream.KeyValue
}

func New(bucket jetstream.KeyValue) *Orders {
	return &Orders{bucket: bucket}
}

func (o *Orders) Keep(ctx context.Context, order acceptedorder.AcceptedOrder) error {
	data, err := yacycrawlcontract.MarshalCrawlOrder(order.CrawlOrder())
	if err != nil {
		return fmt.Errorf("keep order %s: %w", order.OrderID(), err)
	}
	if _, err := o.bucket.Put(ctx, orderKeyOf(order.OrderID()), data); err != nil {
		return fmt.Errorf("keep order %s: %w", order.OrderID(), err)
	}
	return nil
}

func (o *Orders) OrderOf(
	ctx context.Context,
	orderID string,
) (acceptedorder.AcceptedOrder, error) {
	entry, err := o.bucket.Get(ctx, orderKeyOf(orderID))
	if err != nil {
		return acceptedorder.AcceptedOrder{}, fmt.Errorf("read order %s: %w", orderID, err)
	}
	storedOrder, err := yacycrawlcontract.UnmarshalCrawlOrder(entry.Value())
	if err != nil {
		return acceptedorder.AcceptedOrder{}, err
	}
	return acceptedorder.AcceptedOrderFrom(storedOrder)
}

func orderKeyOf(orderID string) string {
	sum := sha256.Sum256([]byte(orderID))
	return hex.EncodeToString(sum[:])
}
