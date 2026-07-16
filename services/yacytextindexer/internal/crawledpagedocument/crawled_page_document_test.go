package crawledpagedocument_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacytextindexer/internal/crawledpagedocument"
)

func TestOfMapsFields(t *testing.T) {
	crawledAt := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	doc := crawledpagedocument.Of(yacycrawlcontract.PageTextRepresentation{
		PageReference: yacycrawlcontract.PageReference{
			CanonicalURL: "https://example.com/",
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
