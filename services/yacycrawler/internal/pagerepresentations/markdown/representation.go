package markdown

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagepublication"
)

type Representation struct{}

func New() Representation {
	return Representation{}
}

func (Representation) Kind() yacycrawlcontract.PageRepresentationKind {
	return yacycrawlcontract.PageRepresentationKindMarkdown
}

func (Representation) ContentFormat() contentformatgraph.Format {
	return contentformatgraph.FormatMarkdown
}

func (Representation) Frame(
	page pagepublication.Page,
	content []byte,
) ([][]byte, error) {
	payload, err := yacycrawlcontract.MarshalPageMarkdownRepresentation(
		yacycrawlcontract.PageMarkdownRepresentation{
			PageReference: page.Reference(),
			Markdown:      content,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"marshal page markdown representation: %w", err,
		)
	}
	return [][]byte{payload}, nil
}
