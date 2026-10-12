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

func TestFlattenLeavesOutAFootnoteMarkerAndTheLinkBackToIt(t *testing.T) {
	text := flattened(
		t,
		`<p>Most traffic uses TCP.<sup id="ref-1"><a href="#note-1">[1]</a></sup> UDP is lighter.</p>`+
			`<ol><li id="note-1"><a href="#ref-1">↑</a> Stevens, TCP/IP Illustrated.</li></ol>`,
	)
	if text != "Most traffic uses TCP. UDP is lighter.\nStevens, TCP/IP Illustrated." {
		t.Fatalf("footnote marks survived: %q", text)
	}
}

func TestFlattenLeavesOutAFootnoteMarkerThatALinkAroundASupMakes(t *testing.T) {
	text := flattened(t, `<p>Most traffic uses TCP.<a href="#fn1" id="fnref1"><sup>1</sup></a></p>`+
		`<ol><li id="fn1">Stevens. <a href="#fnref1">↩︎</a></li></ol>`)
	if text != "Most traffic uses TCP.\nStevens." {
		t.Fatalf("footnote marks survived: %q", text)
	}
}

func TestFlattenKeepsASupThatIsNoFootnoteMarker(t *testing.T) {
	text := flattened(
		t,
		`<p>About 10<sup>6</sup> pages, see <sup><a href="https://example.org/">source</a></sup>.</p>`,
	)
	if text != "About 106 pages, see source." {
		t.Fatalf("a plain sup went missing: %q", text)
	}
}

func TestFlattenKeepsALinkToASectionOfThePage(t *testing.T) {
	text := flattened(
		t,
		`<ul><li><a href="#history">History</a></li></ul><h2 id="history">History</h2>`,
	)
	if text != "History\nHistory" {
		t.Fatalf("a link to a section went missing: %q", text)
	}
}
