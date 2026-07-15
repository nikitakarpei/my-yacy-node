package crawlcapability

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

// PageReference identifies the crawled page a representation was derived from,
// shared by every representation since none of these facts are specific to one.
type PageReference struct {
	CanonicalURL string
	Title        string
	CrawledAt    time.Time
	Language     string
}

func NewPageReference(page CrawledPage) PageReference {
	return PageReference{
		CanonicalURL: page.CanonicalURL,
		Title:        page.Title,
		CrawledAt:    page.CrawledAt,
		Language:     page.Language,
	}
}

type TextRepresentation struct {
	PageReference
	Text []byte
}

type MarkdownRepresentation struct {
	PageReference
	Markdown []byte
}

type RWIRepresentation struct {
	PageReference
	TextLength        int
	WordCount         int
	LocalLinkCount    int
	ExternalLinkCount int
	Postings          []yacymodel.RWIPosting
}
