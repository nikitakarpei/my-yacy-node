package crawlcapability

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
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

type PageFeed interface {
	Representation() yacycrawlcontract.PageRepresentationKind
	ContentFormat() PageContentFormat
	Derive(page CrawledPage, content []byte) (PagePublication, error)
	Publish(ctx context.Context, publication PagePublication) error
}
