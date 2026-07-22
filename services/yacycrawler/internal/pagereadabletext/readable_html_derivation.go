package pagereadabletext

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/htmltext"
)

type ReadableHTMLDerivation struct{}

func NewReadableHTMLDerivation() ReadableHTMLDerivation {
	return ReadableHTMLDerivation{}
}

func (ReadableHTMLDerivation) SourceFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatReadableHTML
}

func (ReadableHTMLDerivation) TargetFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatReadableText
}

func (ReadableHTMLDerivation) Derive(_ string, body []byte) ([]byte, error) {
	text, err := htmltext.Flatten(body)
	if err != nil {
		return nil, err
	}
	return []byte(text), nil
}
