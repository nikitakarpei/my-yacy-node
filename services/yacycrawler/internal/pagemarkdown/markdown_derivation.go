package pagemarkdown

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
) (yacycrawlcontract.PageMarkdownRepresentation, error) {
	markdown, err := render(d.rendering)
	if err != nil {
		return yacycrawlcontract.PageMarkdownRepresentation{}, fmt.Errorf(
			"derive markdown representation: %w", err,
		)
	}
	return yacycrawlcontract.PageMarkdownRepresentation{
		PageReference: page.Reference(),
		Markdown:      markdown,
	}, nil
}
