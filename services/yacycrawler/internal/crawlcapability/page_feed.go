package crawlcapability

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type PublishPage func(ctx context.Context) error

type PageFeed interface {
	Representation() yacycrawlcontract.PageRepresentationKind
	ContentFormat() PageContentFormat
	Derive(page CrawledPage, content []byte) (PublishPage, error)
}
