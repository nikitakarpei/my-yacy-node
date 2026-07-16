package pagemarkdown

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
) (yacycrawlcontract.PageMarkdownRepresentation, error) {
	return yacycrawlcontract.PageMarkdownRepresentation{
		PageReference: page.Reference(),
		Markdown:      content,
	}, nil
}
