package pagetext_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagetext"
)

func TestReadableTextRenderingReadsReadableHTMLAndTargetsReadableText(t *testing.T) {
	rendering := pagetext.NewReadableTextRendering()
	if source := rendering.SourceFormat(); source != crawlcapability.PageContentFormatReadableHTML {
		t.Fatalf("source format = %q, want readable-html", source)
	}
	if format := rendering.Format(); format != crawlcapability.PageContentFormatReadableText {
		t.Fatalf("format = %q, want readable-text", format)
	}
}

func TestReadableTextRenderingFlattensMarkup(t *testing.T) {
	body, err := pagetext.NewReadableTextRendering().Render(
		"https://example.com/",
		[]byte(`<p>first</p><p>second</p>`),
	)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if string(body) != "first\nsecond" {
		t.Fatalf("markup not flattened: %q", body)
	}
}
