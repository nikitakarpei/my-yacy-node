package pagefeed

import (
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type MarkdownFeed struct {
	crawledPageSubject
}

func NewMarkdownFeed(stream jetstream.JetStream, subject string) MarkdownFeed {
	return MarkdownFeed{crawledPageSubject{stream: stream, subject: subject}}
}

func (MarkdownFeed) Representation() yacycrawlcontract.PageRepresentationKind {
	return yacycrawlcontract.PageRepresentationKindMarkdown
}

func (MarkdownFeed) ContentFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatMarkdown
}

func (MarkdownFeed) Frame(
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
