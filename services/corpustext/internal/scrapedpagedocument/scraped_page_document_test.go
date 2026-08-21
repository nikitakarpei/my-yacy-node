package scrapedpagedocument_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/scrapedpagedocument"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
)

func TestOfMapsFields(t *testing.T) {
	scrapedAt := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	doc := scrapedpagedocument.Of(pagescrape.ScrapedPage{
		CanonicalURL: canonicalurltest.CanonicalURLOf(t, "https://example.com/"),
		Title:        "Hi",
		Language:     "en",
		Content:      []byte("words here"),
	}, scrapedAt)
	if doc.URL != "https://example.com/" || doc.Title != "Hi" || doc.Content != "words here" ||
		!doc.CrawledAt.Equal(scrapedAt) || doc.Language != "en" {
		t.Errorf("document = %+v", doc)
	}
}
