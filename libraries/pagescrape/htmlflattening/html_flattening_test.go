package htmlflattening_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/htmlflattening"
)

func TestFlattenStripsMarkup(t *testing.T) {
	text, err := htmlflattening.Flatten(
		[]byte(`<article><h1>Title</h1><p>The quick <b>brown</b> fox.</p></article>`),
	)
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	if strings.Contains(text, "<") {
		t.Fatalf("markup survived: %q", text)
	}
	if !strings.Contains(text, "The quick brown fox.") {
		t.Fatalf("inline markup should not split words: %q", text)
	}
}

func TestFlattenSeparatesBlockElements(t *testing.T) {
	text, err := htmlflattening.Flatten([]byte(`<p>first</p><p>second</p>`))
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	if text != "first\nsecond" {
		t.Fatalf("blocks not separated: %q", text)
	}
}

func TestFlattenCollapsesWhitespaceWithinBlock(t *testing.T) {
	text, err := htmlflattening.Flatten([]byte("<p>The   quick\n  brown\nfox.</p>"))
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	if text != "The quick brown fox." {
		t.Fatalf("whitespace not collapsed within block: %q", text)
	}
}

func TestFlattenDropsScriptAndStyle(t *testing.T) {
	text, err := htmlflattening.Flatten(
		[]byte(`<p>keep</p><script>var drop = 1</script><style>.drop{}</style>`),
	)
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	if strings.Contains(text, "drop") {
		t.Fatalf("script or style content survived: %q", text)
	}
}
