package markdown

import (
	"fmt"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

type DocumentHTMLDerivation struct{}

func NewDocumentHTMLDerivation() DocumentHTMLDerivation {
	return DocumentHTMLDerivation{}
}

func (DocumentHTMLDerivation) SourceFormat() documentextraction.Format {
	return documentextraction.FormatDocumentHTML
}

func (DocumentHTMLDerivation) TargetFormat() documentextraction.Format {
	return documentextraction.FormatMarkdown
}

func (DocumentHTMLDerivation) Derive(
	_ canonicalurl.CanonicalURL,
	body []byte,
) ([]byte, bool, error) {
	markdown, err := htmltomarkdown.ConvertString(string(body))
	if err != nil {
		return nil, false, fmt.Errorf("convert html to markdown: %w", err)
	}
	return []byte(markdown), true, nil
}
