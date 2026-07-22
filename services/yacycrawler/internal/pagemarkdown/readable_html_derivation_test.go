package pagemarkdown_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagemarkdown"
)

func TestDeriveDeclaresReadableHTMLToMarkdown(t *testing.T) {
	derivation := pagemarkdown.NewReadableHTMLDerivation()
	if source := derivation.SourceFormat(); source != crawlcapability.PageContentFormatReadableHTML {
		t.Fatalf("source format = %q, want readable-html", source)
	}
	if target := derivation.TargetFormat(); target != crawlcapability.PageContentFormatMarkdown {
		t.Fatalf("target format = %q, want markdown", target)
	}
}

func TestDeriveConvertsStructureToMarkdown(t *testing.T) {
	body, err := pagemarkdown.NewReadableHTMLDerivation().Derive(
		"https://example.com/",
		[]byte(
			`<h1>Title</h1><p>A <b>bold</b> word and a <a href="http://e.example/x">link</a>.</p>`,
		),
	)
	if err != nil {
		t.Fatalf("derive: %v", err)
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
