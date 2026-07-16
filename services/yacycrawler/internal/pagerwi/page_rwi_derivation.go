package pagerwi

import (
	"fmt"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type Derivation struct {
	text crawlcapability.ContentRendering
}

func NewDerivation(text crawlcapability.ContentRendering) Derivation {
	return Derivation{text: text}
}

func (d Derivation) Accepts(format crawlcapability.PageContentFormat) bool {
	return slices.Contains(d.text.SourceFormats(), format)
}

func (d Derivation) Derive(
	page crawlcapability.CrawledPage,
	render crawlcapability.RenderContent,
) (yacycrawlcontract.PageRWIRepresentation, error) {
	text, err := render(d.text)
	if err != nil {
		return yacycrawlcontract.PageRWIRepresentation{}, fmt.Errorf(
			"render text for rwi: %w",
			err,
		)
	}
	return Build(page, text)
}
