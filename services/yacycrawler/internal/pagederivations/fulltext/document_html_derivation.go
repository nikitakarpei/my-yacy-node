package fulltext

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/htmlflattening"
)

type DocumentHTMLDerivation struct{}

func NewDocumentHTMLDerivation() DocumentHTMLDerivation {
	return DocumentHTMLDerivation{}
}

func (DocumentHTMLDerivation) SourceFormat() contentformatgraph.Format {
	return contentformatgraph.FormatDocumentHTML
}

func (DocumentHTMLDerivation) TargetFormat() contentformatgraph.Format {
	return contentformatgraph.FormatFullText
}

func (DocumentHTMLDerivation) Derive(_ string, body []byte) ([]byte, error) {
	text, err := htmlflattening.Flatten(body)
	if err != nil {
		return nil, err
	}
	return []byte(text), nil
}
