package crawledpagedocument

import (
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func Of(page yacycrawlcontract.PageContentRepresentation) searchdocument.Document {
	return searchdocument.Document{
		Title:     page.Title,
		URL:       page.CanonicalURL,
		Content:   string(page.Body),
		CrawledAt: page.CrawledAt,
		Language:  page.Language,
	}
}
