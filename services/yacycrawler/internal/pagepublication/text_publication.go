package pagepublication

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
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
	representation crawlcapability.TextRepresentation,
) error {
	return publishPageContent(
		ctx,
		p.publisher,
		p.subject,
		representation.PageReference,
		representation.Text,
	)
}
