package jetstream

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/ordersettlement"
)

const (
	msgOrderDecodeFailed = "crawl order decode failed"
	msgOrderTermFailed   = "undeliverable crawl order not terminated"
	msgOrderNakFailed    = "crawl order not returned for redelivery"
)

type OrderReceiver struct {
	deliveries chan ordersettlement.OrderDelivery
}

func NewOrderReceiver(
	ctx context.Context,
	consumer jetstream.Consumer,
) (*OrderReceiver, error) {
	deliveries := make(chan ordersettlement.OrderDelivery)
	consume, err := consumer.Consume(func(msg jetstream.Msg) {
		order, err := yacycrawlcontract.UnmarshalCrawlOrder(msg.Data())
		if err != nil {
			slog.WarnContext(ctx, msgOrderDecodeFailed, slog.Any("error", err))
			if termErr := msg.Term(); termErr != nil {
				slog.WarnContext(ctx, msgOrderTermFailed, slog.Any("error", termErr))
			}
			return
		}
		select {
		case deliveries <- messageDelivery{order: order, message: msg}:
		case <-ctx.Done():
			if nakErr := msg.Nak(); nakErr != nil {
				slog.WarnContext(ctx, msgOrderNakFailed, slog.Any("error", nakErr))
			}
		}
	})
	if err != nil {
		return nil, fmt.Errorf("consume orders: %w", err)
	}
	go func() {
		<-ctx.Done()
		consume.Stop()
	}()
	return &OrderReceiver{deliveries: deliveries}, nil
}

func (r *OrderReceiver) Deliveries() <-chan ordersettlement.OrderDelivery {
	return r.deliveries
}
