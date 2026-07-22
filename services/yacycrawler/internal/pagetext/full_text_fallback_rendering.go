package pagetext

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type FullTextFallbackRendering struct{}

func NewFullTextFallbackRendering() FullTextFallbackRendering {
	return FullTextFallbackRendering{}
}

func (FullTextFallbackRendering) SourceFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatFullText
}

func (FullTextFallbackRendering) Format() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatReadableText
}

func (FullTextFallbackRendering) Render(_ string, body []byte) ([]byte, error) {
	return body, nil
}
