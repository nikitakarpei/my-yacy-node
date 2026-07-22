package readabletext

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/htmlflattening"
)

type ReadableHTMLDerivation struct{}

func NewReadableHTMLDerivation() ReadableHTMLDerivation {
	return ReadableHTMLDerivation{}
}

func (ReadableHTMLDerivation) SourceFormat() contentformatgraph.Format {
	return contentformatgraph.FormatReadableHTML
}

func (ReadableHTMLDerivation) TargetFormat() contentformatgraph.Format {
	return contentformatgraph.FormatReadableText
}

func (ReadableHTMLDerivation) Derive(_ string, body []byte) ([]byte, error) {
	text, err := htmlflattening.Flatten(body)
	if err != nil {
		return nil, err
	}
	return []byte(text), nil
}
