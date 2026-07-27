package crawlrequest

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type OrderStream interface {
	Publish(
		ctx context.Context,
		subject string,
		payload []byte,
		opts ...jetstream.PublishOpt,
	) (*jetstream.PubAck, error)
}

var recallProfile = yacycrawlcontract.CrawlProfile{
	Name:            "corpusrecall",
	Scope:           yacycrawlcontract.ScopeSubpath,
	URLMustMatch:    yacycrawlcontract.MatchAll,
	MaxDepth:        0,
	MaxPagesPerHost: yacycrawlcontract.UnlimitedPagesPerHost,
}

type OrderPlacement struct {
	stream  OrderStream
	subject string
}

func NewOrderPlacement(stream OrderStream, subject string) *OrderPlacement {
	return &OrderPlacement{stream: stream, subject: subject}
}

func (p *OrderPlacement) Place(ctx context.Context, canonicalURL string) error {
	order := yacycrawlcontract.CrawlOrder{
		OrderID:  uuid.NewString(),
		Profile:  recallProfile,
		SeedURLs: []string{canonicalURL},
	}
	payload, err := yacycrawlcontract.MarshalCrawlOrder(order)
	if err != nil {
		return fmt.Errorf("marshal crawl order: %w", err)
	}
	if _, err := p.stream.Publish(ctx, p.subject, payload); err != nil {
		return fmt.Errorf("publish crawl order: %w", err)
	}
	return nil
}
