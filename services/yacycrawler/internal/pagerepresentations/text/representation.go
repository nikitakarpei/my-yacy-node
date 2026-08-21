package text

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagepublication"
)

type Representation struct{}

func New() Representation {
	return Representation{}
}

func (Representation) Kind() yacycrawlcontract.PageRepresentationKind {
	return yacycrawlcontract.PageRepresentationKindText
}

func (Representation) ContentFormat() contentformatgraph.Format {
	return contentformatgraph.FormatReadableText
}

func (Representation) Frame(
	page pagepublication.Page,
	content []byte,
) ([][]byte, error) {
	payload, err := yacycrawlcontract.MarshalPageTextRepresentation(
		yacycrawlcontract.PageTextRepresentation{
			PageReference: page.Reference(),
			Text:          content,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"marshal page text representation: %w", err,
		)
	}
	return [][]byte{payload}, nil
}
