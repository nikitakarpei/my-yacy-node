package pagepublication

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

func pageRepresentationOf(
	format crawlcapability.PageContentFormat,
) yacycrawlcontract.PageRepresentation {
	return yacycrawlcontract.PageRepresentation(format)
}
