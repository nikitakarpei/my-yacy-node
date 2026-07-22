package readabletext_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagederivations/readabletext"
)

func TestReadableHTMLDerivationDeclaresReadableHTMLToReadableText(t *testing.T) {
	derivation := readabletext.NewReadableHTMLDerivation()
	if source := derivation.SourceFormat(); source != contentformatgraph.FormatReadableHTML {
		t.Fatalf("source format = %q, want readable-html", source)
	}
	if target := derivation.TargetFormat(); target != contentformatgraph.FormatReadableText {
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
