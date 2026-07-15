package pagemarkdown

import (
	"fmt"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type Derivation struct {
	rendering Rendering
}

func NewDerivation() Derivation {
	return Derivation{rendering: New()}
}

func (d Derivation) Name() string {
	return string(d.rendering.Format())
}

func (d Derivation) Accepts(format crawlcapability.PageContentFormat) bool {
	return slices.Contains(d.rendering.SourceFormats(), format)
}

func (d Derivation) Derive(
	page crawlcapability.CrawledPage,
	rendered *crawlcapability.RenderedContent,
) (crawlcapability.MarkdownRepresentation, error) {
	markdown, err := rendered.In(d.rendering)
	if err != nil {
		return crawlcapability.MarkdownRepresentation{}, fmt.Errorf(
			"derive markdown representation: %w", err,
		)
	}
	return crawlcapability.MarkdownRepresentation{
		PageReference: crawlcapability.NewPageReference(page),
		Markdown:      markdown,
	}, nil
}
