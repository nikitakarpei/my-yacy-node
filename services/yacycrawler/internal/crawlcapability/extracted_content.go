package crawlcapability

type ExtractedContent struct {
	Title                string
	Body                 []byte
	Format               PageContentFormat
	Language             string
	Links                []string
	LocalLinkCount       int
	ExternalLinkCount    int
	RefusesIndexing      bool
	RefusesLinkDiscovery bool
}
