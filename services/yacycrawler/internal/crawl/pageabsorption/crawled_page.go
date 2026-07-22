package pageabsorption

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
)

type CrawledPage struct {
	CanonicalURL      string
	Title             string
	Body              []byte
	Format            contentformatgraph.Format
	Language          string
	CrawledAt         time.Time
	LocalLinkCount    int
	ExternalLinkCount int
}

func (p CrawledPage) Reference() yacycrawlcontract.PageReference {
	return yacycrawlcontract.PageReference{
		CanonicalURL: p.CanonicalURL,
		Title:        p.Title,
		CrawledAt:    p.CrawledAt,
		Language:     p.Language,
	}
}
