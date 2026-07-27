package markdown

import (
	"fmt"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
)

type DocumentHTMLDerivation struct{}

func NewDocumentHTMLDerivation() DocumentHTMLDerivation {
	return DocumentHTMLDerivation{}
}

func (DocumentHTMLDerivation) SourceFormat() contentformatgraph.Format {
	return contentformatgraph.FormatDocumentHTML
}

func (DocumentHTMLDerivation) TargetFormat() contentformatgraph.Format {
	return contentformatgraph.FormatMarkdown
}

func (DocumentHTMLDerivation) Derive(_ string, body []byte) ([]byte, error) {
	markdown, err := htmltomarkdown.ConvertString(string(body))
	if err != nil {
		return nil, fmt.Errorf("convert html to markdown: %w", err)
	}
	return []byte(markdown), nil
}
