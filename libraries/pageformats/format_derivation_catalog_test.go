package pageformats_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction/bodies/htmltree"
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

const articleWithAFontInline = `<!DOCTYPE html><html lang="en"><head><title>Fonts</title></head>
<body><article><h1>Fonts</h1><p>` + longText + ` alpha<font>beta</font>gamma</p>
<p>` + longText + ` delta<span>epsilon</span>zeta</p></article></body></html>`

const articleWithAVoidElementHoldingChildren = `<!DOCTYPE html><html lang="en">
<head><title>Drawing</title></head><body><nav>navigation menu links elsewhere</nav>
<article><h1>Drawing</h1><p>` + longText + `</p><p>` + longText + `</p>
<svg><source>x</source></svg></article></body></html>`

func bytesIn(
	t *testing.T,
	format documentextraction.Format,
	documentHTML string,
) ([]byte, bool) {
	t.Helper()
	catalog, err := pageformats.New()
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	bodyBytes, derived := catalog.BytesIn(
		t.Context(),
		format,
		documentextraction.Document{
			Format: documentextraction.FormatDocumentHTML,
			Body:   htmlBodyOf(t, documentHTML),
		},
		canonicalurltest.CanonicalURLOf(t, "http://host.example/p"),
	)
	return bodyBytes, derived
}

func htmlBodyOf(t *testing.T, page string) htmltree.Body {
	t.Helper()
	root, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return htmltree.New(root)
}

func TestBytesInRenderTheDocumentBodyInItsOwnFormat(t *testing.T) {
	bodyBytes, derived := bytesIn(t, documentextraction.FormatDocumentHTML, article)
	if !derived {
		t.Fatal("the document body is always available in its own format")
	}
	rendered, err := htmlBodyOf(t, article).Bytes()
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if string(bodyBytes) != string(rendered) {
		t.Fatalf("document-html body was rewritten: %q", bodyBytes)
	}
}

func TestBytesInDerivesMarkdownFromTheDocument(t *testing.T) {
	bodyBytes, derived := bytesIn(t, documentextraction.FormatMarkdown, article)
	if !derived {
		t.Fatal("markdown is derivable from an article")
	}
	markdown := string(bodyBytes)
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

func TestBytesInDerivesReadableTextWithoutTheChrome(t *testing.T) {
	bodyBytes, derived := bytesIn(t, documentextraction.FormatReadableText, article)
	if !derived {
		t.Fatal("readable text is derivable from an article")
	}
	text := string(bodyBytes)
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

func TestBytesInKeepsTheChromeInFullText(t *testing.T) {
	bodyBytes, derived := bytesIn(t, documentextraction.FormatFullText, article)
	if !derived {
		t.Fatal("full text is derivable from any document")
	}
	text := string(bodyBytes)
	if !strings.Contains(text, "navigation menu") {
		t.Fatalf("full text should keep the whole page: %q", text)
	}
}

func TestBytesInFallsBackToFullTextWhenNoArticleIsReadable(t *testing.T) {
	bodyBytes, derived := bytesIn(t, documentextraction.FormatReadableText, pageWithoutAnArticle)
	if !derived {
		t.Fatal("readable text should fall back to the full text of the page")
	}
	if text := string(bodyBytes); !strings.Contains(text, "navigation menu") {
		t.Fatalf("fallback text dropped the page content: %q", text)
	}
}

func TestBytesInDerivesNothingForAFormatNoDerivationProduces(t *testing.T) {
	_, derived := bytesIn(t, documentextraction.Format("audio-transcript"), article)
	if derived {
		t.Fatal("a format no derivation produces should derive nothing")
	}
}

func TestBytesInJoinAFontInlineTheWayTheyJoinASpan(t *testing.T) {
	bodyBytes, derived := bytesIn(t, documentextraction.FormatReadableText, articleWithAFontInline)
	if !derived {
		t.Fatal("readable text is derivable from an article")
	}
	text := string(bodyBytes)
	if !strings.Contains(text, "deltaepsilonzeta") {
		t.Fatalf("a span split the words: %q", text)
	}
	if !strings.Contains(text, "alphabetagamma") {
		t.Fatalf("a font split the words: %q", text)
	}
}

func TestBytesInDeriveNoHTMLFormatForAPageThatCannotRender(t *testing.T) {
	for _, format := range []documentextraction.Format{
		documentextraction.FormatDocumentHTML,
		documentextraction.FormatReadableHTML,
	} {
		if _, derived := bytesIn(t, format, articleWithAVoidElementHoldingChildren); derived {
			t.Errorf("%s derived from a page that cannot render", format)
		}
	}
}

func TestBytesInDeriveTheTextFormatsOfAPageThatCannotRender(t *testing.T) {
	readableText, derived := bytesIn(
		t, documentextraction.FormatReadableText, articleWithAVoidElementHoldingChildren,
	)
	if !derived || !strings.Contains(string(readableText), "quick brown fox") ||
		strings.Contains(string(readableText), "navigation menu") {
		t.Fatalf("readable text not derived from the article: %q", readableText)
	}
	fullText, derived := bytesIn(
		t, documentextraction.FormatFullText, articleWithAVoidElementHoldingChildren,
	)
	if !derived || !strings.Contains(string(fullText), "navigation menu") {
		t.Fatalf("full text not derived from the whole page: %q", fullText)
	}
}
