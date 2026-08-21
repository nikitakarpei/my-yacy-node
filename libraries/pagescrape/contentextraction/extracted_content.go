package contentextraction

import "github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"

type ExtractedContent struct {
	Title                string
	Body                 []byte
	Format               contentformatgraph.Format
	Language             string
	DiscoveredURLs       []string
	LocalLinks           int
	ExternalLinks        int
	RefusesIndexing      bool
	RefusesLinkDiscovery bool
}
