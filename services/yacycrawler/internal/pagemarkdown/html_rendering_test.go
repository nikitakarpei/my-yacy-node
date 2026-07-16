package pagemarkdown_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagemarkdown"
)

func TestHTMLRenderingReadsHTMLAndTargetsMarkdown(t *testing.T) {
	rendering := pagemarkdown.NewHTMLRendering()
	if source := rendering.SourceFormat(); source != crawlcapability.PageContentFormatHTML {
		t.Fatalf("source format = %q, want html", source)
	}
	if format := rendering.Format(); format != crawlcapability.PageContentFormatMarkdown {
		t.Fatalf("format = %q, want markdown", format)
	}
}

func TestHTMLRenderingConvertsStructureToMarkdown(t *testing.T) {
	body, err := pagemarkdown.NewHTMLRendering().Render(
		[]byte(
			`<h1>Title</h1><p>A <b>bold</b> word and a <a href="http://e.example/x">link</a>.</p>`,
		),
	)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	markdown := string(body)
	if !strings.Contains(markdown, "# Title") {
		t.Fatalf("heading not converted: %q", markdown)
	}
	if !strings.Contains(markdown, "**bold**") {
		t.Fatalf("emphasis not converted: %q", markdown)
	}
	if !strings.Contains(markdown, "[link](http://e.example/x)") {
		t.Fatalf("link not converted: %q", markdown)
	}
}
