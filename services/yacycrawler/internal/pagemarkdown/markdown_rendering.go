package pagemarkdown

import (
	"fmt"
	"maps"
	"slices"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

var markdownBySourceFormat = map[crawlcapability.PageContentFormat]func([]byte) ([]byte, error){
	crawlcapability.PageContentFormatHTML: renderMarkdown,
}

type Rendering struct{}

func New() Rendering {
	return Rendering{}
}

func (Rendering) Format() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatMarkdown
}

func (Rendering) SourceFormats() []crawlcapability.PageContentFormat {
	return slices.Sorted(maps.Keys(markdownBySourceFormat))
}

func (Rendering) Render(
	body []byte,
	sourceFormat crawlcapability.PageContentFormat,
) ([]byte, error) {
	render, ok := markdownBySourceFormat[sourceFormat]
	if !ok {
		return nil, fmt.Errorf("markdown rendering cannot accept %s source", sourceFormat)
	}
	return render(body)
}

func renderMarkdown(body []byte) ([]byte, error) {
	markdown, err := htmltomarkdown.ConvertString(string(body))
	if err != nil {
		return nil, fmt.Errorf("convert html to markdown: %w", err)
	}
	return []byte(markdown), nil
}
