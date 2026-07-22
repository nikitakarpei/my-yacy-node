package pagemarkdown

import (
	"fmt"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type ReadableHTMLDerivation struct{}

func NewReadableHTMLDerivation() ReadableHTMLDerivation {
	return ReadableHTMLDerivation{}
}

func (ReadableHTMLDerivation) SourceFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatReadableHTML
}

func (ReadableHTMLDerivation) TargetFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatMarkdown
}

func (ReadableHTMLDerivation) Derive(_ string, body []byte) ([]byte, error) {
	markdown, err := htmltomarkdown.ConvertString(string(body))
	if err != nil {
		return nil, fmt.Errorf("convert html to markdown: %w", err)
	}
	return []byte(markdown), nil
}
