package crawledpagedocument_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/crawledpagedocument"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/canonicalurltest"
)

func TestOfMapsFields(t *testing.T) {
	crawledAt := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	doc := crawledpagedocument.Of(yacycrawlcontract.PageTextRepresentation{
		PageReference: yacycrawlcontract.PageReference{
			CanonicalURL: canonicalurltest.CanonicalURLOf(t, "https://example.com/"),
			Title:        "Hi",
			CrawledAt:    crawledAt,
			Language:     "en",
		},
		Text: []byte("words here"),
	})
	if doc.URL != "https://example.com/" || doc.Title != "Hi" || doc.Content != "words here" ||
		!doc.CrawledAt.Equal(crawledAt) || doc.Language != "en" {
		t.Errorf("document = %+v", doc)
	}
}
