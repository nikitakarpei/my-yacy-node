package contentextraction

import (
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
)

type ExtractedDocument struct {
	Title                string
	Body                 []byte
	Format               contentformatgraph.Format
	Language             string
	DiscoveredURLs       []canonicalurl.CanonicalURL
	LocalLinks           int
	ExternalLinks        int
	RefusesIndexing      bool
	RefusesLinkDiscovery bool
}
