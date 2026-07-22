package readabletext

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
)

type FullTextDerivation struct{}

func NewFullTextDerivation() FullTextDerivation {
	return FullTextDerivation{}
}

func (FullTextDerivation) SourceFormat() contentformatgraph.Format {
	return contentformatgraph.FormatFullText
}

func (FullTextDerivation) TargetFormat() contentformatgraph.Format {
	return contentformatgraph.FormatReadableText
}

func (FullTextDerivation) Derive(_ string, body []byte) ([]byte, error) {
	return body, nil
}
