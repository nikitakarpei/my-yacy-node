package markdown

import (
	"fmt"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
)

type ReadableHTMLDerivation struct{}

func NewReadableHTMLDerivation() ReadableHTMLDerivation {
	return ReadableHTMLDerivation{}
}

func (ReadableHTMLDerivation) SourceFormat() contentformatgraph.Format {
	return contentformatgraph.FormatReadableHTML
}

func (ReadableHTMLDerivation) TargetFormat() contentformatgraph.Format {
	return contentformatgraph.FormatMarkdown
}

func (ReadableHTMLDerivation) Derive(_ string, body []byte) ([]byte, error) {
	markdown, err := htmltomarkdown.ConvertString(string(body))
	if err != nil {
		return nil, fmt.Errorf("convert html to markdown: %w", err)
	}
	return []byte(markdown), nil
}
