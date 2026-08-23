package scrapedpagedocument

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

func Of(
	pageURL canonicalurl.CanonicalURL,
	document documentextraction.Document,
	text []byte,
	scrapedAt time.Time,
) searchdocument.Document {
	return searchdocument.Document{
		Title:     document.Title,
		URL:       pageURL.String(),
		Content:   string(text),
		CrawledAt: scrapedAt,
		Language:  document.Language,
	}
}
