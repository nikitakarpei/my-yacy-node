package pagetext

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/htmltext"
)

type ReadableTextRendering struct{}

func NewReadableTextRendering() ReadableTextRendering {
	return ReadableTextRendering{}
}

func (ReadableTextRendering) SourceFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatReadableHTML
}

func (ReadableTextRendering) Format() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatReadableText
}

func (ReadableTextRendering) Render(_ string, body []byte) ([]byte, error) {
	text, err := htmltext.Flatten(body)
	if err != nil {
		return nil, err
	}
	return []byte(text), nil
}
