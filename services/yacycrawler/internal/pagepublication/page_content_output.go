package pagepublication

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

const pageContentOutputName = "page-content"

type PageContentOutput struct {
	publisher jetstream.JetStream
	subject   string
}

func NewPageContentOutput(publisher jetstream.JetStream, subject string) PageContentOutput {
	return PageContentOutput{publisher: publisher, subject: subject}
}

func (PageContentOutput) Name() string {
	return pageContentOutputName
}

func (o PageContentOutput) Publish(ctx context.Context, page crawlcapability.ExtractedPage) error {
	payload, err := yacycrawlcontract.MarshalPageContentRepresentation(
		yacycrawlcontract.PageContentRepresentation{
			CanonicalURL: page.CanonicalURL,
			Title:        page.Title,
			CrawledAt:    page.FetchedAt,
			Language:     page.Language,
			Format:       yacycrawlcontract.PageFormatText,
			Body:         []byte(page.Text),
		},
	)
	if err != nil {
		return fmt.Errorf("marshal page content representation: %w", err)
	}
	if _, err := o.publisher.Publish(ctx, o.subject, payload); err != nil {
		return classifyPublishError(err)
	}
	return nil
}
