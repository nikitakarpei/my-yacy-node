package readabletext_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/pagederivations/readabletext"
)

func TestReadableHTMLDerivationDeclaresReadableHTMLToReadableText(t *testing.T) {
	derivation := readabletext.NewReadableHTMLDerivation()
	if source := derivation.SourceFormat(); source != documentextraction.FormatReadableHTML {
		t.Fatalf("source format = %q, want readable-html", source)
	}
	if target := derivation.TargetFormat(); target != documentextraction.FormatReadableText {
		t.Fatalf("target format = %q, want readable-text", target)
	}
}

func TestReadableHTMLDerivationFlattensMarkup(t *testing.T) {
	body, derived, err := readabletext.NewReadableHTMLDerivation().Derive(
		canonicalurltest.CanonicalURLOf(t, "https://example.com/"),
		[]byte(`<p>first</p><p>second</p>`),
	)
	if err != nil || !derived {
		t.Fatalf("derive: derived=%v err=%v", derived, err)
	}
	if string(body) != "first\nsecond" {
		t.Fatalf("markup not flattened: %q", body)
	}
}
