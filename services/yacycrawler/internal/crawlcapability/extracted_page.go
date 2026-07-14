package crawlcapability

import "time"

type ExtractedPage struct {
	CanonicalURL      string
	Title             string
	Body              []byte
	Format            PageContentFormat
	Language          string
	CrawledAt         time.Time
	LocalLinkCount    int
	ExternalLinkCount int
}
