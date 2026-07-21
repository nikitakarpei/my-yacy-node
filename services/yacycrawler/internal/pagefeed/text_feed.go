package pagefeed

import (
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type TextFeed struct {
	crawledPageSubject
}

func NewTextFeed(stream jetstream.JetStream, subject string) TextFeed {
	return TextFeed{crawledPageSubject{stream: stream, subject: subject}}
}

func (TextFeed) Representation() yacycrawlcontract.PageRepresentationKind {
	return yacycrawlcontract.PageRepresentationKindText
}

func (TextFeed) ContentFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatReadableText
}

func (TextFeed) Derive(
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
