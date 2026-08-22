package readabletext_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagederivations/readabletext"
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
	body, err := readabletext.NewReadableHTMLDerivation().Derive(
		"https://example.com/",
		[]byte(`<p>first</p><p>second</p>`),
	)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if string(body) != "first\nsecond" {
		t.Fatalf("markup not flattened: %q", body)
	}
}
