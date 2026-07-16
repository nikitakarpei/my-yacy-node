package pagetext

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type TextRendering struct{}

func NewTextRendering() TextRendering {
	return TextRendering{}
}

func (TextRendering) SourceFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatText
}

func (TextRendering) Format() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatText
}

func (TextRendering) Render(body []byte) ([]byte, error) {
	return body, nil
}
