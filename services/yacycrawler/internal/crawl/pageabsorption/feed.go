package pageabsorption

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
)

// PagePublication is what one page feed sends for a page, opaque outside the feed that derived it.
type PagePublication struct {
	messages [][]byte
}

func NewPagePublication(messages ...[]byte) PagePublication {
	return PagePublication{messages: messages}
}

func (p PagePublication) Messages() [][]byte {
	return p.messages
}

type Feed interface {
	Representation() yacycrawlcontract.PageRepresentationKind
	ContentFormat() contentformatgraph.Format
	Frame(page CrawledPage, content []byte) (PagePublication, error)
	Publish(ctx context.Context, publication PagePublication) error
}
