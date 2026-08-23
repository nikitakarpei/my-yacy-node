package pageformats_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
)

const longText = "The quick brown fox jumps over the lazy dog while the industrious " +
	"beaver builds a sturdy dam across the wide and winding river near the old mill town."

const article = `<!DOCTYPE html><html lang="en"><head><title>Sample Article</title></head>
<body><nav>navigation menu links elsewhere</nav>
<article><h1>Sample Article</h1><p>` + longText + `</p><p>` + longText + `</p></article>
</body></html>`

const pageWithoutAnArticle = `<!DOCTYPE html><html lang="en"><head><title>Index</title></head>
<body><nav>navigation menu links elsewhere</nav></body></html>`

func bodyIn(
	t *testing.T,
	format documentextraction.Format,
	documentHTML string,
) ([]byte, bool) {
	t.Helper()
	derivableFormats, err := pageformats.New()
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	body, derived, err := derivableFormats.BodyIn(
		format,
		documentextraction.Document{
			Format: documentextraction.FormatDocumentHTML,
			Body:   []byte(documentHTML),
		},
		canonicalurltest.CanonicalURLOf(t, "http://host.example/p"),
	)
	if err != nil {
		t.Fatalf("body in %s: %v", format, err)
	}
	return body, derived
}

func TestBodyInReturnsTheDocumentBodyInItsOwnFormat(t *testing.T) {
	body, derived := bodyIn(t, documentextraction.FormatDocumentHTML, article)
	if !derived {
		t.Fatal("the document body is always available in its own format")
	}
	if string(body) != article {
		t.Fatalf("document-html body was rewritten: %q", body)
	}
}

func TestBodyInDerivesMarkdownFromTheDocument(t *testing.T) {
	body, derived := bodyIn(t, documentextraction.FormatMarkdown, article)
	if !derived {
		t.Fatal("markdown is derivable from an article")
	}
	markdown := string(body)
	if !strings.Contains(markdown, "quick brown fox") {
		t.Fatalf("article text dropped: %q", markdown)
	}
	if strings.Contains(markdown, "<p>") {
		t.Fatalf("html survived the conversion to markdown: %q", markdown)
	}
	if strings.Contains(markdown, "navigation menu") {
		t.Fatalf("chrome should be stripped before markdown: %q", markdown)
	}
}

func TestBodyInDerivesReadableTextWithoutTheChrome(t *testing.T) {
	body, derived := bodyIn(t, documentextraction.FormatReadableText, article)
	if !derived {
		t.Fatal("readable text is derivable from an article")
	}
	text := string(body)
	if !strings.Contains(text, "quick brown fox") {
		t.Fatalf("article text dropped: %q", text)
	}
	if strings.Contains(text, "navigation menu") {
		t.Fatalf("chrome should be stripped from readable text: %q", text)
	}
	if strings.Contains(text, "<") {
		t.Fatalf("markup survived: %q", text)
	}
}

func TestBodyInKeepsTheChromeInFullText(t *testing.T) {
	body, derived := bodyIn(t, documentextraction.FormatFullText, article)
	if !derived {
		t.Fatal("full text is derivable from any document")
	}
	text := string(body)
	if !strings.Contains(text, "navigation menu") {
		t.Fatalf("full text should keep the whole page: %q", text)
	}
}

func TestBodyInFallsBackToFullTextWhenNoArticleIsReadable(t *testing.T) {
	body, derived := bodyIn(t, documentextraction.FormatReadableText, pageWithoutAnArticle)
	if !derived {
		t.Fatal("readable text should fall back to the full text of the page")
	}
	if text := string(body); !strings.Contains(text, "navigation menu") {
		t.Fatalf("fallback text dropped the page content: %q", text)
	}
}

func TestBodyInDerivesNothingForAFormatNoDerivationProduces(t *testing.T) {
	_, derived := bodyIn(t, documentextraction.Format("audio-transcript"), article)
	if derived {
		t.Fatal("a format no derivation produces should derive nothing")
	}
}
