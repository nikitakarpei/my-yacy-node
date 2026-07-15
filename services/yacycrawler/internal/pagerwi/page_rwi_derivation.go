package pagerwi

import (
	"fmt"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

const representationName = "rwi"

type Derivation struct {
	text crawlcapability.ContentRendering
}

func NewDerivation(text crawlcapability.ContentRendering) Derivation {
	return Derivation{text: text}
}

func (Derivation) Name() string {
	return representationName
}

func (d Derivation) Accepts(format crawlcapability.PageContentFormat) bool {
	return slices.Contains(d.text.SourceFormats(), format)
}

func (d Derivation) Derive(
	page crawlcapability.CrawledPage,
	rendered *crawlcapability.RenderedContent,
) (crawlcapability.RWIRepresentation, error) {
	text, err := rendered.In(d.text)
	if err != nil {
		return crawlcapability.RWIRepresentation{}, fmt.Errorf("render text for rwi: %w", err)
	}
	return Build(page, text)
}
