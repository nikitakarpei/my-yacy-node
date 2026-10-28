package pagefeedreading

import (
	"context"
	"encoding/xml"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkresolution"
)

const (
	rdfNamespace  = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
	atomNamespace = "http://www.w3.org/2005/Atom"
)

var (
	rssRoot  = xml.Name{Local: "rss"}
	rdfRoot  = xml.Name{Space: rdfNamespace, Local: "RDF"}
	atomRoot = xml.Name{Space: atomNamespace, Local: "feed"}
)

type feedDocument interface {
	linksFrom(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		linkResolution LinkResolution,
	) []linkresolution.Link
}

func feedDocumentFor(root xml.Name) (feedDocument, bool) {
	switch root {
	case rssRoot:
		return &rssDocument{}, true
	case rdfRoot:
		return &rdfDocument{}, true
	case atomRoot:
		return &atomDocument{}, true
	default:
		return nil, false
	}
}

func linksAgainst(baseURL canonicalurl.CanonicalURL, references []string) []linkresolution.Link {
	links := make([]linkresolution.Link, 0, len(references))
	for _, reference := range references {
		if strings.TrimSpace(reference) != "" {
			links = append(links, linkresolution.Link{BaseURL: baseURL, Reference: reference})
		}
	}
	return links
}
