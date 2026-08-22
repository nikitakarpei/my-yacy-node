package scrapedpagedocument

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

func Of(page pagescrape.ScrapedPage, scrapedAt time.Time) searchdocument.Document {
	return searchdocument.Document{
		Title:     page.Title,
		URL:       page.CanonicalURL.String(),
		Content:   string(page.Content),
		CrawledAt: scrapedAt,
		Language:  page.Language,
	}
}
