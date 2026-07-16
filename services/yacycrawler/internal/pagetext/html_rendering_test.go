package pagetext_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagetext"
)

func TestHTMLRenderingReadsHTMLAndTargetsText(t *testing.T) {
	rendering := pagetext.NewHTMLRendering()
	if source := rendering.SourceFormat(); source != crawlcapability.PageContentFormatHTML {
		t.Fatalf("source format = %q, want html", source)
	}
	if format := rendering.Format(); format != crawlcapability.PageContentFormatText {
		t.Fatalf("format = %q, want text", format)
	}
}

func TestHTMLRenderingStripsMarkup(t *testing.T) {
	body, err := pagetext.NewHTMLRendering().Render(
		[]byte(`<article><h1>Title</h1><p>The quick <b>brown</b> fox.</p></article>`),
	)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	text := string(body)
	if strings.Contains(text, "<") {
		t.Fatalf("markup survived: %q", text)
	}
	if !strings.Contains(text, "The quick brown fox.") {
		t.Fatalf("inline markup should not split words: %q", text)
	}
}

func TestHTMLRenderingSeparatesBlockElements(t *testing.T) {
	body, err := pagetext.NewHTMLRendering().Render([]byte(`<p>first</p><p>second</p>`))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if string(body) != "first\nsecond" {
		t.Fatalf("blocks not separated: %q", body)
	}
}

func TestHTMLRenderingCollapsesWhitespaceWithinBlock(t *testing.T) {
	body, err := pagetext.NewHTMLRendering().Render([]byte("<p>The   quick\n  brown\nfox.</p>"))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if string(body) != "The quick brown fox." {
		t.Fatalf("whitespace not collapsed within block: %q", body)
	}
}

func TestHTMLRenderingDropsScriptAndStyle(t *testing.T) {
	body, err := pagetext.NewHTMLRendering().Render(
		[]byte(`<p>keep</p><script>var drop = 1</script><style>.drop{}</style>`),
	)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(string(body), "drop") {
		t.Fatalf("script or style content survived: %q", body)
	}
}
