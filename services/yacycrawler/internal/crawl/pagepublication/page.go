package pagepublication

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type Page struct {
	CanonicalURL  string
	Title         string
	Body          []byte
	Format        contentformatgraph.Format
	Language      string
	CrawledAt     time.Time
	LocalLinks    int
	ExternalLinks int
}

func (p Page) Reference() yacycrawlcontract.PageReference {
	return yacycrawlcontract.PageReference{
		CanonicalURL: p.CanonicalURL,
		Title:        p.Title,
		CrawledAt:    p.CrawledAt,
		Language:     p.Language,
	}
}
