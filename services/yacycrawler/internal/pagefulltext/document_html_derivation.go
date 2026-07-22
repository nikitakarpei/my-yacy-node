package pagefulltext

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/htmltext"
)

type DocumentHTMLDerivation struct{}

func NewDocumentHTMLDerivation() DocumentHTMLDerivation {
	return DocumentHTMLDerivation{}
}

func (DocumentHTMLDerivation) SourceFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatDocumentHTML
}

func (DocumentHTMLDerivation) TargetFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatFullText
}

func (DocumentHTMLDerivation) Derive(_ string, body []byte) ([]byte, error) {
	text, err := htmltext.Flatten(body)
	if err != nil {
		return nil, err
	}
	return []byte(text), nil
}
