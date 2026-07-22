package contentextraction

import "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"

type ExtractedContent struct {
	Title                string
	Body                 []byte
	Format               contentformatgraph.Format
	Language             string
	Links                []string
	LocalLinkCount       int
	ExternalLinkCount    int
	RefusesIndexing      bool
	RefusesLinkDiscovery bool
}
