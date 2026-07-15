package pagemarkdown_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagemarkdown"
)

func TestDerivationNameIsMarkdown(t *testing.T) {
	if name := pagemarkdown.NewDerivation().Name(); name != "markdown" {
		t.Fatalf("name = %q, want markdown", name)
	}
}

func TestDerivationAcceptsOnlyRenderingSourceFormats(t *testing.T) {
	derivation := pagemarkdown.NewDerivation()
	if !derivation.Accepts(crawlcapability.PageContentFormatHTML) {
		t.Fatal("should accept html")
	}
	if derivation.Accepts(crawlcapability.PageContentFormatText) {
		t.Fatal("should not accept text")
	}
}

func TestDerivationProducesMarkdownRepresentation(t *testing.T) {
	page := crawlcapability.CrawledPage{
		CanonicalURL: "http://example.com/a",
		Title:        "Hi",
		Body:         []byte("<h1>hello</h1>"),
		Format:       crawlcapability.PageContentFormatHTML,
		Language:     "en",
		CrawledAt:    time.Unix(1_700_000_000, 0),
	}
	rendered := crawlcapability.NewRenderedContent(page.Body, page.Format)
	representation, err := pagemarkdown.NewDerivation().Derive(page, rendered)
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
