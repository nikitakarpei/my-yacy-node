package pagepublication

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type PageRepresentation interface {
	Kind() yacycrawlcontract.PageRepresentationKind
	ContentFormat() contentformatgraph.Format
	Frame(page Page, content []byte) ([][]byte, error)
	Publish(ctx context.Context, messages [][]byte) error
}
