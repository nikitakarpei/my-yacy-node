package pagerwi_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagerwi"
)

type identityRendering struct{}

func (identityRendering) Format() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatText
}

func (identityRendering) SourceFormats() []crawlcapability.PageContentFormat {
	return []crawlcapability.PageContentFormat{crawlcapability.PageContentFormatHTML}
}

func (identityRendering) Render(
	body []byte,
	_ crawlcapability.PageContentFormat,
) ([]byte, error) {
	return body, nil
}

func TestDerivationNameIsRWI(t *testing.T) {
	if name := pagerwi.NewDerivation(identityRendering{}).Name(); name != "rwi" {
		t.Fatalf("name = %q, want rwi", name)
	}
}

func TestDerivationAcceptsItsTextRenderingsSourceFormats(t *testing.T) {
	derivation := pagerwi.NewDerivation(identityRendering{})
	if !derivation.Accepts(crawlcapability.PageContentFormatHTML) {
		t.Fatal("should accept html, since the injected rendering does")
	}
	if derivation.Accepts(crawlcapability.PageContentFormatMarkdown) {
		t.Fatal("should not accept markdown")
	}
}

func TestDerivationDerivesFromRenderedText(t *testing.T) {
	page := samplePage()
	page.Body = []byte(sampleText)
	page.Format = crawlcapability.PageContentFormatHTML
	rendered := crawlcapability.NewRenderedContent(page.Body, page.Format)
	representation, err := pagerwi.NewDerivation(identityRendering{}).Derive(page, rendered)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(representation.Postings) == 0 {
		t.Fatal("no postings")
	}
}
