// Package jetstream places crawl orders on the orders stream.
package jetstream

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

var recallProfile = yacycrawlcontract.CrawlProfile{
	Name:            "corpusrecall",
	Scope:           yacycrawlcontract.ScopeSubpath,
	URLMustMatch:    yacycrawlcontract.MatchAll,
	MaxDepth:        0,
	MaxPagesPerHost: yacycrawlcontract.UnlimitedPagesPerHost,
}

type CrawlOrderPlacement struct {
	orders  jetstream.JetStream
	subject string
}

func NewCrawlOrderPlacement(
	orders jetstream.JetStream,
	subject string,
) *CrawlOrderPlacement {
	return &CrawlOrderPlacement{orders: orders, subject: subject}
}

func (p *CrawlOrderPlacement) Place(
	ctx context.Context,
	canonicalURL yacycrawlcontract.CanonicalURL,
) error {
	order := yacycrawlcontract.CrawlOrder{
		OrderID:  uuid.NewString(),
		Profile:  recallProfile,
		SeedURLs: []yacycrawlcontract.CanonicalURL{canonicalURL},
	}
	payload, err := yacycrawlcontract.MarshalCrawlOrder(order)
	if err != nil {
		return fmt.Errorf("marshal crawl order: %w", err)
	}
	if _, err := p.orders.Publish(ctx, p.subject, payload); err != nil {
		return fmt.Errorf("publish crawl order: %w", err)
	}
	return nil
}
