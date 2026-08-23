package readablehtml_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/pagederivations/readablehtml"
)

const longText = "The quick brown fox jumps over the lazy dog while the industrious " +
	"beaver builds a sturdy dam across the wide and winding river near the old mill town."

const document = `<!DOCTYPE html><html lang="en"><head><title>Sample Article</title></head>
<body><nav>navigation menu links elsewhere</nav>
<article><h1>Sample Article</h1><p>` + longText + `</p><p>` + longText + `</p></article>
</body></html>`

func TestDeriveDeclaresDocumentHTMLToReadableHTML(t *testing.T) {
	derivation := readablehtml.FromDocumentHTML()
	if source := derivation.SourceFormat(); source != documentextraction.FormatDocumentHTML {
		t.Fatalf("source format = %q, want document-html", source)
	}
	if target := derivation.TargetFormat(); target != documentextraction.FormatReadableHTML {
		t.Fatalf("target format = %q, want readable-html", target)
	}
}

func TestDeriveExtractsMainArticle(t *testing.T) {
	body, derived, err := readablehtml.FromDocumentHTML().BodyFrom(
		t.Context(),
		canonicalurltest.CanonicalURLOf(t, "http://host.example/p"), []byte(document),
	)
	if err != nil || !derived {
		t.Fatalf("derive: derived=%v err=%v", derived, err)
	}
	readable := string(body)
	if !strings.Contains(readable, "quick brown fox") {
		t.Fatalf("main content dropped: %q", readable)
	}
	if strings.Contains(readable, "navigation menu") {
		t.Fatalf("chrome should be stripped from readable html: %q", readable)
	}
}

func TestAPageWithNoArticleDerivesNothing(t *testing.T) {
	_, derived, err := readablehtml.FromDocumentHTML().BodyFrom(
		t.Context(),
		canonicalurltest.CanonicalURLOf(t, "http://host.example/p"),
		[]byte("<html><body></body></html>"),
	)
	if err != nil {
		t.Fatalf("an empty page is not a failure, got %v", err)
	}
	if derived {
		t.Fatal("a page with no article should derive nothing")
	}
}
