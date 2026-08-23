package readabletext

import (
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/htmlflattening"
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

func (ReadableHTMLDerivation) Derive(
	_ canonicalurl.CanonicalURL,
	body []byte,
) ([]byte, bool, error) {
	text, err := htmlflattening.Flatten(body)
	if err != nil {
		return nil, false, err
	}
	return []byte(text), true, nil
}
