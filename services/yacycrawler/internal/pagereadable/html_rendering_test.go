package pagereadable_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagereadable"
)

const longText = "The quick brown fox jumps over the lazy dog while the industrious " +
	"beaver builds a sturdy dam across the wide and winding river near the old mill town."

const document = `<!DOCTYPE html><html lang="en"><head><title>Sample Article</title></head>
<body><nav>navigation menu links elsewhere</nav>
<article><h1>Sample Article</h1><p>` + longText + `</p><p>` + longText + `</p></article>
</body></html>`

func TestHTMLRenderingReadsDocumentHTMLAndTargetsReadableHTML(t *testing.T) {
	rendering := pagereadable.NewHTMLRendering()
	if source := rendering.SourceFormat(); source != crawlcapability.PageContentFormatDocumentHTML {
		t.Fatalf("source format = %q, want document-html", source)
	}
	if format := rendering.Format(); format != crawlcapability.PageContentFormatReadableHTML {
		t.Fatalf("format = %q, want readable-html", format)
	}
}

func TestHTMLRenderingExtractsMainArticle(t *testing.T) {
	body, err := pagereadable.NewHTMLRendering().
		Render("http://host.example/p", []byte(document))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	readable := string(body)
	if !strings.Contains(readable, "quick brown fox") {
		t.Fatalf("main content dropped: %q", readable)
	}
	if strings.Contains(readable, "navigation menu") {
		t.Fatalf("chrome should be stripped from readable html: %q", readable)
	}
}

func TestHTMLRenderingEmptyContentUnextractable(t *testing.T) {
	_, err := pagereadable.NewHTMLRendering().
		Render("http://host.example/p", []byte("<html><body></body></html>"))
	if !errors.Is(err, crawlcapability.ErrUnextractable) {
		t.Fatalf("want ErrUnextractable, got %v", err)
	}
}
