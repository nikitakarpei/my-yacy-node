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
) (crawlcapability.PublishPage, error) {
	payload, err := yacycrawlcontract.MarshalPageMarkdownRepresentation(
		yacycrawlcontract.PageMarkdownRepresentation{
			PageReference: page.Reference(),
			Markdown:      content,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("marshal page markdown representation: %w", err)
	}
	return func(ctx context.Context) error {
		if _, err := f.publisher.Publish(ctx, f.subject, payload); err != nil {
			return classifyPublishError(err)
		}
		return nil
	}, nil
}
