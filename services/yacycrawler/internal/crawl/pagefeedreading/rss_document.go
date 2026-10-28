package pagefeedreading

import (
	"context"
	"encoding/xml"
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkresolution"
)

const guidNotPermaLink = "false"

var webSchemes = []string{"http", "https"}

type rssDocument struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Links []rssLink `xml:"link"`
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Links []rssLink `xml:"link"`
	GUID  rssGUID   `xml:"guid"`
}

type rssLink struct {
	XMLName   xml.Name
	Reference string `xml:",chardata"`
}

type rssGUID struct {
	Reference   string `xml:",chardata"`
	IsPermaLink string `xml:"isPermaLink,attr"`
}

func (document *rssDocument) linksFrom(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ LinkResolution,
) []linkresolution.Link {
	references := rssReferencesOf(document.Channel.Links)
	for _, item := range document.Channel.Items {
		references = append(references, item.references()...)
	}
	return linksAgainst(pageURL, references)
}

func rssReferencesOf(links []rssLink) []string {
	var references []string
	for _, link := range links {
		if link.XMLName.Space == "" {
			references = append(references, link.Reference)
		}
	}
	return references
}

func (item rssItem) references() []string {
	if references := rssReferencesOf(item.Links); len(references) > 0 {
		return references
	}
	if item.GUID.isPermaLink() {
		return []string{item.GUID.Reference}
	}
	return nil
}

func (guid rssGUID) isPermaLink() bool {
	if guid.IsPermaLink == guidNotPermaLink {
		return false
	}
	address, err := url.Parse(strings.TrimSpace(guid.Reference))
	if err != nil {
		return false
	}
	for _, scheme := range webSchemes {
		if strings.EqualFold(address.Scheme, scheme) {
			return true
		}
	}
	return false
}
