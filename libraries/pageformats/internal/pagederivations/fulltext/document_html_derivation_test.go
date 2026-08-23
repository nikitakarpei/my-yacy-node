package fulltext_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/pagederivations/fulltext"
)

func TestDeriveFlattensWholeDocumentAndStripsMarkup(t *testing.T) {
	derivation := fulltext.FromDocumentHTML()
	body := []byte(
		`<html><body><nav>navigation menu</nav>` +
			`<article><p>the quick fox</p></article>` +
			`<script>var drop = 1</script></body></html>`,
	)
	text, derived, err := derivation.BodyFrom(
		t.Context(),
		canonicalurltest.CanonicalURLOf(t, "http://example.com/"), body,
	)
	if err != nil || !derived {
		t.Fatalf("derive: derived=%v err=%v", derived, err)
	}
	flat := string(text)
	if !strings.Contains(flat, "navigation menu") {
		t.Fatal("content outside the article should survive whole-document flattening")
	}
	if !strings.Contains(flat, "the quick fox") {
		t.Fatal("article content should survive flattening")
	}
	if strings.Contains(flat, "drop") {
		t.Fatal("script content should be stripped")
	}
}

func TestDeriveDeclaresDocumentHTMLToFullText(t *testing.T) {
	derivation := fulltext.FromDocumentHTML()
	if derivation.SourceFormat() != documentextraction.FormatDocumentHTML {
		t.Fatalf("source format = %q", derivation.SourceFormat())
	}
	if derivation.TargetFormat() != documentextraction.FormatFullText {
		t.Fatalf("target format = %q", derivation.TargetFormat())
	}
}
