package fulltext

import (
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/htmlflattening"
)

type DocumentHTMLDerivation struct{}

func NewDocumentHTMLDerivation() DocumentHTMLDerivation {
	return DocumentHTMLDerivation{}
}

func (DocumentHTMLDerivation) SourceFormat() documentextraction.Format {
	return documentextraction.FormatDocumentHTML
}

func (DocumentHTMLDerivation) TargetFormat() documentextraction.Format {
	return documentextraction.FormatFullText
}

func (DocumentHTMLDerivation) Derive(_ string, body []byte) ([]byte, error) {
	text, err := htmlflattening.Flatten(body)
	if err != nil {
		return nil, err
	}
	return []byte(text), nil
}
