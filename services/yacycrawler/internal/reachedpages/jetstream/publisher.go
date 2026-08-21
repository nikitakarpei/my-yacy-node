// Package jetstream publishes pages the crawler reached.
package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type Publisher struct {
	stream jetstream.JetStream
}

func New(stream jetstream.JetStream) *Publisher {
	return &Publisher{stream: stream}
}

func (p *Publisher) Publish(ctx context.Context, canonicalURL string) error {
	data, err := yacycrawlcontract.MarshalReachedPage(
		yacycrawlcontract.ReachedPage{CanonicalURL: canonicalURL},
	)
	if err != nil {
		return fmt.Errorf("marshal reached page %s: %w", canonicalURL, err)
	}
	if _, err := p.stream.Publish(ctx, yacycrawlcontract.ReachedPageSubject, data); err != nil {
		return fmt.Errorf("publish reached page %s: %w", canonicalURL, err)
	}
	return nil
}
