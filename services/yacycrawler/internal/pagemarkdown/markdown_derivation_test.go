package pagemarkdown_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagemarkdown"
)

func TestDeriveTargetsMarkdownFormat(t *testing.T) {
	if format := pagemarkdown.New().Format(); format != crawlcapability.PageContentFormatMarkdown {
		t.Fatalf("format = %q, want markdown", format)
	}
}

func TestDeriveConvertsHTMLStructureToMarkdown(t *testing.T) {
	body, err := pagemarkdown.New().Derive(
		[]byte(
			`<h1>Title</h1><p>A <b>bold</b> word and a <a href="http://e.example/x">link</a>.</p>`,
		),
		crawlcapability.PageContentFormatHTML,
	)
	if err != nil {
		t.Fatalf("Derive: %v", err)
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

func TestDerivePassesMarkdownThrough(t *testing.T) {
	body, err := pagemarkdown.New().Derive(
		[]byte("# already markdown"),
		crawlcapability.PageContentFormatMarkdown,
	)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if string(body) != "# already markdown" {
		t.Fatalf("markdown body = %q", body)
	}
}

func TestSourceFormatsDeclaresHTMLAndMarkdown(t *testing.T) {
	want := []crawlcapability.PageContentFormat{
		crawlcapability.PageContentFormatHTML,
		crawlcapability.PageContentFormatMarkdown,
	}
	if got := pagemarkdown.New().SourceFormats(); !slices.Equal(got, want) {
		t.Fatalf("SourceFormats() = %v, want %v", got, want)
	}
}
