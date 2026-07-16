package pagetext

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
) (yacycrawlcontract.PageTextRepresentation, error) {
	return yacycrawlcontract.PageTextRepresentation{
		PageReference: page.Reference(),
		Text:          content,
	}, nil
}
