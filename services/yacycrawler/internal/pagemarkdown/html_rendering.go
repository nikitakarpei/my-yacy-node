package pagemarkdown

import (
	"fmt"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type HTMLRendering struct{}

func NewHTMLRendering() HTMLRendering {
	return HTMLRendering{}
}

func (HTMLRendering) SourceFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatReadableHTML
}

func (HTMLRendering) Format() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatMarkdown
}

func (HTMLRendering) Render(_ string, body []byte) ([]byte, error) {
	markdown, err := htmltomarkdown.ConvertString(string(body))
	if err != nil {
		return nil, fmt.Errorf("convert html to markdown: %w", err)
	}
	return []byte(markdown), nil
}
