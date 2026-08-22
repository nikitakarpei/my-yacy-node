package readabletext

import (
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/htmlflattening"
)

type ReadableHTMLDerivation struct{}

func NewReadableHTMLDerivation() ReadableHTMLDerivation {
	return ReadableHTMLDerivation{}
}

func (ReadableHTMLDerivation) SourceFormat() documentextraction.Format {
	return documentextraction.FormatReadableHTML
}

func (ReadableHTMLDerivation) TargetFormat() documentextraction.Format {
	return documentextraction.FormatReadableText
}

func (ReadableHTMLDerivation) Derive(_ string, body []byte) ([]byte, error) {
	text, err := htmlflattening.Flatten(body)
	if err != nil {
		return nil, err
	}
	return []byte(text), nil
}
