package pagepublication

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
)

type PageRepresentation interface {
	Kind() yacycrawlcontract.PageRepresentationKind
	ContentFormat() contentformatgraph.Format
	Frame(page Page, content []byte) ([][]byte, error)
	Publish(ctx context.Context, messages [][]byte) error
}
