package pagefeedreading

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkresolution"
)

const relationAlternate = "alternate"

type atomDocument struct {
	BaseURL string      `xml:"http://www.w3.org/XML/1998/namespace base,attr"`
	Links   []atomLink  `xml:"http://www.w3.org/2005/Atom link"`
	Entries []atomEntry `xml:"http://www.w3.org/2005/Atom entry"`
}

type atomEntry struct {
	BaseURL string     `xml:"http://www.w3.org/XML/1998/namespace base,attr"`
	Links   []atomLink `xml:"http://www.w3.org/2005/Atom link"`
}

type atomLink struct {
	BaseURL   string `xml:"http://www.w3.org/XML/1998/namespace base,attr"`
	Relation  string `xml:"rel,attr"`
	Reference string `xml:"href,attr"`
}

func (document *atomDocument) linksFrom(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	linkResolution LinkResolution,
) []linkresolution.Link {
	feedBaseURL := linkResolution.BaseURLOf(ctx, pageURL, document.BaseURL)
	links := alternateLinksOf(ctx, document.Links, feedBaseURL, linkResolution)
	for _, entry := range document.Entries {
		entryBaseURL := linkResolution.BaseURLOf(ctx, feedBaseURL, entry.BaseURL)
		links = append(links, alternateLinksOf(ctx, entry.Links, entryBaseURL, linkResolution)...)
	}
	return links
}

func alternateLinksOf(
	ctx context.Context,
	atomLinks []atomLink,
	enclosingBaseURL canonicalurl.CanonicalURL,
	linkResolution LinkResolution,
) []linkresolution.Link {
	var links []linkresolution.Link
	for _, link := range atomLinks {
		if link.Reference == "" || (link.Relation != "" && link.Relation != relationAlternate) {
			continue
		}
		links = append(links, linkresolution.Link{
			BaseURL:   linkResolution.BaseURLOf(ctx, enclosingBaseURL, link.BaseURL),
			Reference: link.Reference,
		})
	}
	return links
}
