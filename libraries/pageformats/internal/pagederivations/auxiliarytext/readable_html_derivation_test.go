package auxiliarytext_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction/bodies/htmltree"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/pagederivations/auxiliarytext"
)

func htmlBodyOf(t *testing.T, page string) htmltree.Body {
	t.Helper()
	root, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return htmltree.New(root)
}

func TestReadableHTMLDerivationDeclaresReadableHTMLToAuxiliaryText(t *testing.T) {
	derivation := auxiliarytext.FromReadableHTML()
	if source := derivation.SourceFormat(); source != documentextraction.FormatReadableHTML {
		t.Fatalf("source format = %q, want readable-html", source)
	}
	if target := derivation.TargetFormat(); target != documentextraction.FormatAuxiliaryText {
		t.Fatalf("target format = %q, want auxiliary-text", target)
	}
}

func TestReadableHTMLDerivationHoldsTheNotesWithoutFootnoteMarks(t *testing.T) {
	body, derived, err := auxiliarytext.FromReadableHTML().BodyFrom(
		t.Context(),
		canonicalurltest.CanonicalURLOf(t, "https://example.com/"),
		htmlBodyOf(t, `<p>TCP.<sup id="ref-1"><a href="#note-1">[1]</a></sup></p>`+
			`<ol><li id="note-1"><a href="#ref-1">↑</a> Stevens.</li></ol>`),
	)
	if err != nil || !derived {
		t.Fatalf("derive: derived=%v err=%v", derived, err)
	}
	bodyBytes, err := body.Bytes()
	if err != nil {
		t.Fatalf("bytes: %v", err)
	}
	if auxiliaryText := string(bodyBytes); auxiliaryText != "Stevens." {
		t.Fatalf("auxiliary text = %q", auxiliaryText)
	}
}
