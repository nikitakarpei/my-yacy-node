package markdown

import (
	"fmt"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

type ReadableHTMLDerivation struct{}

func NewReadableHTMLDerivation() ReadableHTMLDerivation {
	return ReadableHTMLDerivation{}
}

func (ReadableHTMLDerivation) SourceFormat() documentextraction.Format {
	return documentextraction.FormatReadableHTML
}

func (ReadableHTMLDerivation) TargetFormat() documentextraction.Format {
	return documentextraction.FormatMarkdown
}

func (ReadableHTMLDerivation) Derive(
	_ canonicalurl.CanonicalURL,
	body []byte,
) ([]byte, bool, error) {
	markdown, err := htmltomarkdown.ConvertString(string(body))
	if err != nil {
		return nil, false, fmt.Errorf("convert html to markdown: %w", err)
	}
	return []byte(markdown), true, nil
}
