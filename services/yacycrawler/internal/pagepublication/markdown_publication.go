package pagepublication

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type MarkdownPublication struct {
	publisher jetstream.JetStream
	subject   string
}

func NewMarkdownPublication(publisher jetstream.JetStream, subject string) MarkdownPublication {
	return MarkdownPublication{publisher: publisher, subject: subject}
}

func (p MarkdownPublication) Publish(
	ctx context.Context,
	representation yacycrawlcontract.PageMarkdownRepresentation,
) error {
	payload, err := yacycrawlcontract.MarshalPageMarkdownRepresentation(representation)
	if err != nil {
		return fmt.Errorf("marshal page markdown representation: %w", err)
	}
	if _, err := p.publisher.Publish(ctx, p.subject, payload); err != nil {
		return classifyPublishError(err)
	}
	return nil
}
