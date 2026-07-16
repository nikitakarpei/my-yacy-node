package pagerwi

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type Derivation struct{}

func NewDerivation() Derivation {
	return Derivation{}
}

func (Derivation) Derive(
	page crawlcapability.CrawledPage,
	content []byte,
) (yacycrawlcontract.PageRWIRepresentation, error) {
	return Build(page, content)
}
