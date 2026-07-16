package crawlcapability

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type PublishPage func(ctx context.Context) error

type PageFeed interface {
	Representation() yacycrawlcontract.PageRepresentationKind
	Accepts(format PageContentFormat) bool
	Derive(page CrawledPage, render RenderContent) (PublishPage, error)
}
