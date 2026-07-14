package pagemarkdown

import (
	"fmt"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type MarkdownDerivation struct{}

func New() MarkdownDerivation {
	return MarkdownDerivation{}
}

func (MarkdownDerivation) Format() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatMarkdown
}

func (MarkdownDerivation) SourceFormats() []crawlcapability.PageContentFormat {
	return []crawlcapability.PageContentFormat{
		crawlcapability.PageContentFormatHTML,
		crawlcapability.PageContentFormatMarkdown,
	}
}

func (MarkdownDerivation) Derive(
	body []byte,
	sourceFormat crawlcapability.PageContentFormat,
) ([]byte, error) {
	if sourceFormat != crawlcapability.PageContentFormatHTML {
		return body, nil
	}
	markdown, err := htmltomarkdown.ConvertString(string(body))
	if err != nil {
		return nil, fmt.Errorf("convert html to markdown: %w", err)
	}
	return []byte(markdown), nil
}
