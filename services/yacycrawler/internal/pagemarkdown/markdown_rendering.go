package pagemarkdown

import (
	"fmt"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type Rendering struct{}

func New() Rendering {
	return Rendering{}
}

func (Rendering) Format() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatMarkdown
}

func (Rendering) SourceFormats() []crawlcapability.PageContentFormat {
	return []crawlcapability.PageContentFormat{
		crawlcapability.PageContentFormatHTML,
		crawlcapability.PageContentFormatMarkdown,
	}
}

func (Rendering) Render(
	body []byte,
	sourceFormat crawlcapability.PageContentFormat,
) ([]byte, error) {
	switch sourceFormat {
	case crawlcapability.PageContentFormatMarkdown:
		return body, nil
	case crawlcapability.PageContentFormatHTML:
		markdown, err := htmltomarkdown.ConvertString(string(body))
		if err != nil {
			return nil, fmt.Errorf("convert html to markdown: %w", err)
		}
		return []byte(markdown), nil
	}
	return nil, fmt.Errorf("markdown rendering cannot accept %s source", sourceFormat)
}
