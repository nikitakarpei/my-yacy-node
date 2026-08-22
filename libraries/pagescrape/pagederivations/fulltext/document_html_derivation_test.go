package fulltext_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagederivations/fulltext"
)

func TestDeriveFlattensWholeDocumentAndStripsMarkup(t *testing.T) {
	derivation := fulltext.NewDocumentHTMLDerivation()
	body := []byte(
		`<html><body><nav>navigation menu</nav>` +
			`<article><p>the quick fox</p></article>` +
			`<script>var drop = 1</script></body></html>`,
	)
	text, err := derivation.Derive("http://example.com/", body)
	if err != nil {
		t.Fatal(err)
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
	derivation := fulltext.NewDocumentHTMLDerivation()
	if derivation.SourceFormat() != documentextraction.FormatDocumentHTML {
		t.Fatalf("source format = %q", derivation.SourceFormat())
	}
	if derivation.TargetFormat() != documentextraction.FormatFullText {
		t.Fatalf("target format = %q", derivation.TargetFormat())
	}
}
