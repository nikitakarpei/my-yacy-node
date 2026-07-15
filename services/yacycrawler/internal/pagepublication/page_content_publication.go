package pagepublication

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

func publishPageContent(
	ctx context.Context,
	publisher jetstream.JetStream,
	subject string,
	reference crawlcapability.PageReference,
	body []byte,
) error {
	payload, err := yacycrawlcontract.MarshalPageContentRepresentation(
		yacycrawlcontract.PageContentRepresentation{
			CanonicalURL: reference.CanonicalURL,
			Title:        reference.Title,
			CrawledAt:    reference.CrawledAt,
			Language:     reference.Language,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("marshal page content representation: %w", err)
	}
	if _, err := publisher.Publish(ctx, subject, payload); err != nil {
		return classifyPublishError(err)
	}
	return nil
}
