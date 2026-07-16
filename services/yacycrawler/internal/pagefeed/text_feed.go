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
) (crawlcapability.PagePublication, error) {
	payload, err := yacycrawlcontract.MarshalPageTextRepresentation(
		yacycrawlcontract.PageTextRepresentation{
			PageReference: page.Reference(),
			Text:          content,
		},
	)
	if err != nil {
		return crawlcapability.PagePublication{}, fmt.Errorf(
			"marshal page text representation: %w", err,
		)
	}
	return crawlcapability.NewPagePublication(payload), nil
}

func (f TextFeed) Publish(ctx context.Context, publication crawlcapability.PagePublication) error {
	for _, message := range publication.Messages() {
		if _, err := f.publisher.Publish(ctx, f.subject, message); err != nil {
			return classifyPublishError(err)
		}
	}
	return nil
}
