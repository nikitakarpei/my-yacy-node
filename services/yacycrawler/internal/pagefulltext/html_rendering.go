package pagefulltext

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/htmltext"
)

type HTMLRendering struct{}

func NewHTMLRendering() HTMLRendering {
	return HTMLRendering{}
}

func (HTMLRendering) SourceFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatDocumentHTML
}

func (HTMLRendering) Format() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatFullText
}

func (HTMLRendering) Render(_ string, body []byte) ([]byte, error) {
	text, err := htmltext.Flatten(body)
	if err != nil {
		return nil, err
	}
	return []byte(text), nil
}
