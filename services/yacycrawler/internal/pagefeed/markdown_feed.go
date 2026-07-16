package pagefeed

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type MarkdownFeed struct {
	publisher     jetstream.JetStream
	subject       string
	contentFormat crawlcapability.PageContentFormat
}

func NewMarkdownFeed(
	publisher jetstream.JetStream,
	subject string,
	contentFormat crawlcapability.PageContentFormat,
) MarkdownFeed {
	return MarkdownFeed{publisher: publisher, subject: subject, contentFormat: contentFormat}
}

func (MarkdownFeed) Representation() yacycrawlcontract.PageRepresentationKind {
	return yacycrawlcontract.PageRepresentationKindMarkdown
}

func (f MarkdownFeed) ContentFormat() crawlcapability.PageContentFormat {
	return f.contentFormat
}

func (f MarkdownFeed) Derive(
	page crawlcapability.CrawledPage,
	content []byte,
) (crawlcapability.PagePublication, error) {
	payload, err := yacycrawlcontract.MarshalPageMarkdownRepresentation(
		yacycrawlcontract.PageMarkdownRepresentation{
			PageReference: page.Reference(),
			Markdown:      content,
		},
	)
	if err != nil {
		return crawlcapability.PagePublication{}, fmt.Errorf(
			"marshal page markdown representation: %w", err,
		)
	}
	return crawlcapability.NewPagePublication(payload), nil
}

func (f MarkdownFeed) Publish(
	ctx context.Context,
	publication crawlcapability.PagePublication,
) error {
	for _, message := range publication.Messages() {
		if _, err := f.publisher.Publish(ctx, f.subject, message); err != nil {
			return classifyPublishError(err)
		}
	}
	return nil
}
