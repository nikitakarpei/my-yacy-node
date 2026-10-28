package pagefeedreading

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkresolution"
)

type rdfDocument struct {
	Channel rdfLinked   `xml:"http://purl.org/rss/1.0/ channel"`
	Items   []rdfLinked `xml:"http://purl.org/rss/1.0/ item"`
}

type rdfLinked struct {
	Link string `xml:"http://purl.org/rss/1.0/ link"`
}

func (document *rdfDocument) linksFrom(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ LinkResolution,
) []linkresolution.Link {
	references := make([]string, 0, 1+len(document.Items))
	references = append(references, document.Channel.Link)
	for _, item := range document.Items {
		references = append(references, item.Link)
	}
	return linksAgainst(pageURL, references)
}
