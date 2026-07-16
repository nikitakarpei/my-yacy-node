package pagefeed

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type TextFeed struct {
	publisher     jetstream.JetStream
	subject       string
	contentFormat crawlcapability.PageContentFormat
}

func NewTextFeed(
	publisher jetstream.JetStream,
	subject string,
	contentFormat crawlcapability.PageContentFormat,
) TextFeed {
	return TextFeed{publisher: publisher, subject: subject, contentFormat: contentFormat}
}

func (TextFeed) Representation() yacycrawlcontract.PageRepresentationKind {
	return yacycrawlcontract.PageRepresentationKindText
}

func (f TextFeed) ContentFormat() crawlcapability.PageContentFormat {
	return f.contentFormat
}

func (f TextFeed) Derive(
	page crawlcapability.CrawledPage,
	content []byte,
) (crawlcapability.PublishPage, error) {
	payload, err := yacycrawlcontract.MarshalPageTextRepresentation(
		yacycrawlcontract.PageTextRepresentation{
			PageReference: page.Reference(),
			Text:          content,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("marshal page text representation: %w", err)
	}
	return func(ctx context.Context) error {
		if _, err := f.publisher.Publish(ctx, f.subject, payload); err != nil {
			return classifyPublishError(err)
		}
		return nil
	}, nil
}
