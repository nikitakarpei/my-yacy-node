package markdown_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/pagederivations/markdown"
)

func TestDeriveDeclaresDocumentHTMLToMarkdown(t *testing.T) {
	derivation := markdown.FromDocumentHTML()
	if source := derivation.SourceFormat(); source != documentextraction.FormatDocumentHTML {
		t.Fatalf("source format = %q, want document-html", source)
	}
	if target := derivation.TargetFormat(); target != documentextraction.FormatMarkdown {
		t.Fatalf("target format = %q, want markdown", target)
	}
}

func TestDeriveConvertsWholeDocumentToMarkdown(t *testing.T) {
	body, derived, err := markdown.FromDocumentHTML().BodyFrom(
		t.Context(),
		canonicalurltest.CanonicalURLOf(t, "https://example.com/"),
		[]byte(
			`<html><body><h1>Title</h1><p>A <b>bold</b> word and a `+
				`<a href="http://e.example/x">link</a>.</p></body></html>`,
		),
	)
	if err != nil || !derived {
		t.Fatalf("derive: derived=%v err=%v", derived, err)
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
