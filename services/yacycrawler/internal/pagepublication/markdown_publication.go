package pagepublication

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
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
	representation crawlcapability.MarkdownRepresentation,
) error {
	return publishPageContent(
		ctx, p.publisher, p.subject, representation.PageReference, representation.Markdown,
	)
}
