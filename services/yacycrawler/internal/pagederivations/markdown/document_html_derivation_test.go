package markdown_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagederivations/markdown"
)

func TestDeriveDeclaresDocumentHTMLToMarkdown(t *testing.T) {
	derivation := markdown.NewDocumentHTMLDerivation()
	if source := derivation.SourceFormat(); source != contentformatgraph.FormatDocumentHTML {
		t.Fatalf("source format = %q, want document-html", source)
	}
	if target := derivation.TargetFormat(); target != contentformatgraph.FormatMarkdown {
		t.Fatalf("target format = %q, want markdown", target)
	}
}

func TestDeriveConvertsWholeDocumentToMarkdown(t *testing.T) {
	body, err := markdown.NewDocumentHTMLDerivation().Derive(
		"https://example.com/",
		[]byte(
			`<html><body><h1>Title</h1><p>A <b>bold</b> word and a `+
				`<a href="http://e.example/x">link</a>.</p></body></html>`,
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
