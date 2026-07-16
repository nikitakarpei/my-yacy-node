package pagetext

import (
	"fmt"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type Derivation struct {
	rendering Rendering
}

func NewDerivation(rendering Rendering) Derivation {
	return Derivation{rendering: rendering}
}

func (d Derivation) Accepts(format crawlcapability.PageContentFormat) bool {
	return slices.Contains(d.rendering.SourceFormats(), format)
}

func (d Derivation) Derive(
	page crawlcapability.CrawledPage,
	render crawlcapability.RenderContent,
) (yacycrawlcontract.PageTextRepresentation, error) {
	text, err := render(d.rendering)
	if err != nil {
		return yacycrawlcontract.PageTextRepresentation{}, fmt.Errorf(
			"derive text representation: %w",
			err,
		)
	}
	return yacycrawlcontract.PageTextRepresentation{
		PageReference: page.Reference(),
		Text:          text,
	}, nil
}
