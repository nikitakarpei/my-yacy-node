package pagepublication

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
)

type PublicationProgress interface {
	PageDisposed(reason disposal.Reason)
	PagePublished(representation yacycrawlcontract.PageRepresentationKind)
	PublicationWaited()
}
