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
