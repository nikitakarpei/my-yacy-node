package readabletext_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction/bodies/htmltree"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/pagederivations/readabletext"
)

func htmlBodyOf(t *testing.T, page string) htmltree.Body {
	t.Helper()
	root, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return htmltree.New(root)
}

func bytesOf(t *testing.T, body documentextraction.Body) string {
	t.Helper()
	bodyBytes, err := body.Bytes()
	if err != nil {
		t.Fatalf("bytes: %v", err)
	}
	return string(bodyBytes)
}

func TestReadableHTMLDerivationDeclaresReadableHTMLToReadableText(t *testing.T) {
	derivation := readabletext.FromReadableHTML()
	if source := derivation.SourceFormat(); source != documentextraction.FormatReadableHTML {
		t.Fatalf("source format = %q, want readable-html", source)
	}
	if target := derivation.TargetFormat(); target != documentextraction.FormatReadableText {
		t.Fatalf("target format = %q, want readable-text", target)
	}
}

func TestReadableHTMLDerivationFlattensMarkup(t *testing.T) {
	body, derived, err := readabletext.FromReadableHTML().BodyFrom(
		t.Context(),
		canonicalurltest.CanonicalURLOf(t, "https://example.com/"),
		htmlBodyOf(t, `<p>first</p><p>second</p>`),
	)
	if err != nil || !derived {
		t.Fatalf("derive: derived=%v err=%v", derived, err)
	}
	if readableText := bytesOf(t, body); readableText != "first\nsecond" {
		t.Fatalf("markup not flattened: %q", readableText)
	}
}

func TestReadableHTMLDerivationLeavesOutFootnoteMarks(t *testing.T) {
	body, derived, err := readabletext.FromReadableHTML().BodyFrom(
		t.Context(),
		canonicalurltest.CanonicalURLOf(t, "https://example.com/"),
		htmlBodyOf(t, `<p>TCP.<sup id="ref-1"><a href="#note-1">[1]</a></sup></p>`+
			`<ol><li id="note-1"><a href="#ref-1">↑</a> Stevens.</li></ol>`),
	)
	if err != nil || !derived {
		t.Fatalf("derive: derived=%v err=%v", derived, err)
	}
	if readableText := bytesOf(t, body); readableText != "TCP.\nStevens." {
		t.Fatalf("footnote marks survived: %q", readableText)
	}
}

func TestReadableHTMLDerivationLeavesOutPermalinks(t *testing.T) {
	body, derived, err := readabletext.FromReadableHTML().BodyFrom(
		t.Context(),
		canonicalurltest.CanonicalURLOf(t, "https://example.com/"),
		htmlBodyOf(t, `<p id="section-1">TCP is reliable.<a href="#section-1">¶</a></p>`),
	)
	if err != nil || !derived {
		t.Fatalf("derive: derived=%v err=%v", derived, err)
	}
	if readableText := bytesOf(t, body); readableText != "TCP is reliable." {
		t.Fatalf("permalink survived: %q", readableText)
	}
}

func TestReadableHTMLDerivationLeavesOutMediaWikiChrome(t *testing.T) {
	body, derived, err := readabletext.FromReadableHTML().BodyFrom(
		t.Context(),
		canonicalurltest.CanonicalURLOf(t, "https://example.com/"),
		htmlBodyOf(
			t,
			`<h2>History<span class="mw-editsection">[<a href="/edit">edit</a>]</span></h2>`+
				`<p>TCP is reliable.</p>`,
		),
	)
	if err != nil || !derived {
		t.Fatalf("derive: derived=%v err=%v", derived, err)
	}
	if readableText := bytesOf(t, body); readableText != "History\nTCP is reliable." {
		t.Fatalf("MediaWiki chrome survived: %q", readableText)
	}
}
