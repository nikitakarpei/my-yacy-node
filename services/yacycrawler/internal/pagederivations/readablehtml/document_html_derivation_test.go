package readablehtml_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagederivations/readablehtml"
)

const longText = "The quick brown fox jumps over the lazy dog while the industrious " +
	"beaver builds a sturdy dam across the wide and winding river near the old mill town."

const document = `<!DOCTYPE html><html lang="en"><head><title>Sample Article</title></head>
<body><nav>navigation menu links elsewhere</nav>
<article><h1>Sample Article</h1><p>` + longText + `</p><p>` + longText + `</p></article>
</body></html>`

func TestDeriveDeclaresDocumentHTMLToReadableHTML(t *testing.T) {
	derivation := readablehtml.NewDocumentHTMLDerivation()
	if source := derivation.SourceFormat(); source != contentformatgraph.FormatDocumentHTML {
		t.Fatalf("source format = %q, want document-html", source)
	}
	if target := derivation.TargetFormat(); target != contentformatgraph.FormatReadableHTML {
		t.Fatalf("target format = %q, want readable-html", target)
	}
}

func TestDeriveExtractsMainArticle(t *testing.T) {
	body, err := readablehtml.NewDocumentHTMLDerivation().
		Derive("http://host.example/p", []byte(document))
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	readable := string(body)
	if !strings.Contains(readable, "quick brown fox") {
		t.Fatalf("main content dropped: %q", readable)
	}
	if strings.Contains(readable, "navigation menu") {
		t.Fatalf("chrome should be stripped from readable html: %q", readable)
	}
}

func TestDeriveEmptyContentUnextractable(t *testing.T) {
	_, err := readablehtml.NewDocumentHTMLDerivation().
		Derive("http://host.example/p", []byte("<html><body></body></html>"))
	if !errors.Is(err, contentformatgraph.ErrUnderivable) {
		t.Fatalf("want ErrUnextractable, got %v", err)
	}
}
