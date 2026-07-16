package pagemarkdown_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagemarkdown"
)

func TestDerivationProducesMarkdownRepresentation(t *testing.T) {
	page := crawlcapability.CrawledPage{
		CanonicalURL: "http://example.com/a",
		Title:        "Hi",
		Format:       crawlcapability.PageContentFormatHTML,
		Language:     "en",
		CrawledAt:    time.Unix(1_700_000_000, 0),
	}
	representation, err := pagemarkdown.NewDerivation().Derive(page, []byte("# hello"))
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if representation.CanonicalURL != page.CanonicalURL || representation.Title != page.Title {
		t.Fatalf("page reference not carried over: %+v", representation.PageReference)
	}
	if string(representation.Markdown) != "# hello" {
		t.Fatalf("markdown = %q", representation.Markdown)
	}
}
