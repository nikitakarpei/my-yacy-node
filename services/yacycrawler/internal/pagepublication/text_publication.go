package pagepublication

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type TextPublication struct {
	publisher jetstream.JetStream
	subject   string
}

func NewTextPublication(publisher jetstream.JetStream, subject string) TextPublication {
	return TextPublication{publisher: publisher, subject: subject}
}

func (p TextPublication) Publish(
	ctx context.Context,
	representation yacycrawlcontract.PageTextRepresentation,
) error {
	payload, err := yacycrawlcontract.MarshalPageTextRepresentation(representation)
	if err != nil {
		return fmt.Errorf("marshal page text representation: %w", err)
	}
	if _, err := p.publisher.Publish(ctx, p.subject, payload); err != nil {
		return classifyPublishError(err)
	}
	return nil
}
