package pagepublication

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
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
