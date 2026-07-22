package pagereadabletext

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type FullTextDerivation struct{}

func NewFullTextDerivation() FullTextDerivation {
	return FullTextDerivation{}
}

func (FullTextDerivation) SourceFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatFullText
}

func (FullTextDerivation) TargetFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatReadableText
}

func (FullTextDerivation) Derive(_ string, body []byte) ([]byte, error) {
	return body, nil
}
