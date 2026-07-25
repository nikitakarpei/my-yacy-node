package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type messageDelivery struct {
	order   yacycrawlcontract.CrawlOrder
	message jetstream.Msg
}

func (d messageDelivery) Order() yacycrawlcontract.CrawlOrder {
	return d.order
}

func (d messageDelivery) Acknowledge(context.Context) error {
	if err := d.message.Ack(); err != nil {
		return fmt.Errorf("acknowledge order: %w", err)
	}
	return nil
}

func (d messageDelivery) Return(context.Context) error {
	if err := d.message.Nak(); err != nil {
		return fmt.Errorf("return order for redelivery: %w", err)
	}
	return nil
}

func (d messageDelivery) ExtendOwnership(context.Context) error {
	if err := d.message.InProgress(); err != nil {
		return fmt.Errorf("extend order ownership: %w", err)
	}
	return nil
}
