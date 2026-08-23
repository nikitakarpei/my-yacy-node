package scrapedpagedocument_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/scrapedpagedocument"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

func TestOfMapsFields(t *testing.T) {
	scrapedAt := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	doc := scrapedpagedocument.Of(
		canonicalurltest.CanonicalURLOf(t, "https://example.com/"),
		documentextraction.Document{Title: "Hi", Language: "en"},
		[]byte("words here"),
		scrapedAt,
	)
	if doc.URL != "https://example.com/" || doc.Title != "Hi" || doc.Content != "words here" ||
		!doc.CrawledAt.Equal(scrapedAt) || doc.Language != "en" {
		t.Errorf("document = %+v", doc)
	}
}
