package fulltext

import (
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/htmlflattening"
)

type DocumentHTMLDerivation struct{}

func FromDocumentHTML() DocumentHTMLDerivation {
	return DocumentHTMLDerivation{}
}

func (DocumentHTMLDerivation) SourceFormat() documentextraction.Format {
	return documentextraction.FormatDocumentHTML
}

func (DocumentHTMLDerivation) TargetFormat() documentextraction.Format {
	return documentextraction.FormatFullText
}

func (DocumentHTMLDerivation) BodyFrom(
	_ canonicalurl.CanonicalURL,
	body []byte,
) ([]byte, bool, error) {
	text, err := htmlflattening.Flatten(body)
	if err != nil {
		return nil, false, err
	}
	return []byte(text), true, nil
}
