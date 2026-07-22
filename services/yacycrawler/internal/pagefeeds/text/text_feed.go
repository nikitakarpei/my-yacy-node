package text

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
)

type Feed struct{}

func New() Feed {
	return Feed{}
}

func (Feed) Representation() yacycrawlcontract.PageRepresentationKind {
	return yacycrawlcontract.PageRepresentationKindText
}

func (Feed) ContentFormat() contentformatgraph.Format {
	return contentformatgraph.FormatReadableText
}

func (Feed) Frame(
	page pageabsorption.CrawledPage,
	content []byte,
) (pageabsorption.PagePublication, error) {
	payload, err := yacycrawlcontract.MarshalPageTextRepresentation(
		yacycrawlcontract.PageTextRepresentation{
			PageReference: page.Reference(),
			Text:          content,
		},
	)
	if err != nil {
		return pageabsorption.PagePublication{}, fmt.Errorf(
			"marshal page text representation: %w", err,
		)
	}
	return pageabsorption.NewPagePublication(payload), nil
}
