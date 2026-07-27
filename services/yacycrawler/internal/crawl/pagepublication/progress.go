package pagepublication

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type PublicationProgress interface {
	PagePublished(representation yacycrawlcontract.PageRepresentationKind)
	PublicationWaited()
}
