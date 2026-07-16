package pagetext_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagetext"
)

func TestDerivationAcceptsOnlyRenderingSourceFormats(t *testing.T) {
	derivation := pagetext.NewDerivation(pagetext.New())
	if !derivation.Accepts(crawlcapability.PageContentFormatHTML) {
		t.Fatal("should accept html")
	}
	if derivation.Accepts(crawlcapability.PageContentFormatMarkdown) {
		t.Fatal("should not accept markdown")
	}
}

func TestDerivationProducesTextRepresentation(t *testing.T) {
	page := crawlcapability.CrawledPage{
		CanonicalURL: "http://example.com/a",
		Title:        "Hi",
		Body:         []byte("<p>hello</p>"),
		Format:       crawlcapability.PageContentFormatHTML,
		Language:     "en",
		CrawledAt:    time.Unix(1_700_000_000, 0),
	}
	rendered := renderPage(page)
	representation, err := pagetext.NewDerivation(pagetext.New()).Derive(page, rendered)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if representation.CanonicalURL != page.CanonicalURL || representation.Title != page.Title {
		t.Fatalf("page reference not carried over: %+v", representation.PageReference)
	}
	if string(representation.Text) != "hello" {
		t.Fatalf("text = %q", representation.Text)
	}
}

func renderPage(page crawlcapability.CrawledPage) crawlcapability.RenderContent {
	return func(rendering crawlcapability.ContentRendering) ([]byte, error) {
		return rendering.Render(page.Body, page.Format)
	}
}
