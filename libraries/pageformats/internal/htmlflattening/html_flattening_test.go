package htmlflattening_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/htmlflattening"
)

func flattened(t *testing.T, page string) string {
	t.Helper()
	root, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return htmlflattening.Flatten(root)
}

func TestFlattenStripsMarkup(t *testing.T) {
	text := flattened(t, `<article><h1>Title</h1><p>The quick <b>brown</b> fox.</p></article>`)
	if strings.Contains(text, "<") {
		t.Fatalf("markup survived: %q", text)
	}
	if !strings.Contains(text, "The quick brown fox.") {
		t.Fatalf("inline markup should not split words: %q", text)
	}
}

func TestFlattenSeparatesBlockElements(t *testing.T) {
	if text := flattened(t, `<p>first</p><p>second</p>`); text != "first\nsecond" {
		t.Fatalf("blocks not separated: %q", text)
	}
}

func TestFlattenSeparatesTheTitleFromTheBody(t *testing.T) {
	text := flattened(
		t,
		`<html><head><title>Hi</title></head><body><p>alpha beta</p></body></html>`,
	)
	if text != "Hi\nalpha beta" {
		t.Fatalf("title runs into the body text: %q", text)
	}
}

func TestFlattenKeepsInlineMarkupFromSplittingAWord(t *testing.T) {
	if text := flattened(t, `<p>hyper<em>text</em></p>`); text != "hypertext" {
		t.Fatalf("inline markup split the word: %q", text)
	}
}

func TestFlattenCollapsesWhitespaceWithinBlock(t *testing.T) {
	if text := flattened(t, "<p>The   quick\n  brown\nfox.</p>"); text != "The quick brown fox." {
		t.Fatalf("whitespace not collapsed within block: %q", text)
	}
}

func TestFlattenDropsScriptAndStyle(t *testing.T) {
	text := flattened(t, `<p>keep</p><script>var drop = 1</script><style>.drop{}</style>`)
	if strings.Contains(text, "drop") {
		t.Fatalf("script or style content survived: %q", text)
	}
}
