package pagefulltext_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefulltext"
)

func TestRenderFlattensWholeDocumentAndStripsMarkup(t *testing.T) {
	rendering := pagefulltext.NewHTMLRendering()
	body := []byte(
		`<html><body><nav>navigation menu</nav>` +
			`<article><p>the quick fox</p></article>` +
			`<script>var drop = 1</script></body></html>`,
	)
	text, err := rendering.Render("http://example.com/", body)
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

func TestRenderDeclaresDocumentHTMLToFullText(t *testing.T) {
	rendering := pagefulltext.NewHTMLRendering()
	if rendering.SourceFormat() != crawlcapability.PageContentFormatDocumentHTML {
		t.Fatalf("source format = %q", rendering.SourceFormat())
	}
	if rendering.Format() != crawlcapability.PageContentFormatFullText {
		t.Fatalf("format = %q", rendering.Format())
	}
}
