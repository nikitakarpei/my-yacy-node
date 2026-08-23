package documentextraction_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

const article = `<!DOCTYPE html><html lang="en"><head><title>Sample Article</title></head>
<body><article><h1>Sample Article</h1>
<p>` + longText + `</p>
<p>` + longText + `</p>
<a href="/local/page">local</a>
<a href="http://other.example/ext">external</a>
</article></body></html>`

const longText = "The quick brown fox jumps over the lazy dog while the industrious " +
	"beaver builds a sturdy dam across the wide and winding river near the old mill town."

func TestExtractArticle(t *testing.T) {
	doc, err := documentextraction.DocumentFrom(
		t.Context(),
		[]byte(article),
		"text/html",
		canonicalurltest.CanonicalURLOf(t, "http://host.example/dir/p"),
	)
	if err != nil {
		t.Fatalf("DocumentFrom: %v", err)
	}
	if doc.Title != "Sample Article" {
		t.Fatalf("title = %q", doc.Title)
	}
	if doc.Format != documentextraction.FormatDocumentHTML {
		t.Fatalf("format = %q, want document-html", doc.Format)
	}
	if !strings.Contains(string(doc.Body), "quick brown fox") {
		t.Fatalf("body missing article content: %q", doc.Body)
	}
	if !strings.Contains(string(doc.Body), "<p") {
		t.Fatalf("body should keep article markup: %q", doc.Body)
	}
	if doc.Language != "en" {
		t.Fatalf("language = %q, want en", doc.Language)
	}
	if doc.LocalLinks != 1 || doc.ExternalLinks != 1 {
		t.Fatalf("links local=%d external=%d", doc.LocalLinks, doc.ExternalLinks)
	}
}

func TestExtractYieldsWholeDocument(t *testing.T) {
	doc, err := documentextraction.DocumentFrom(
		t.Context(),
		[]byte(article),
		"text/html",
		canonicalurltest.CanonicalURLOf(t, "http://host.example/dir/p"),
	)
	if err != nil {
		t.Fatalf("DocumentFrom: %v", err)
	}
	body := string(doc.Body)
	if !strings.Contains(body, "<title>Sample Article</title>") {
		t.Fatalf("whole document should retain head markup: %q", body)
	}
}

func TestMediaTypesDeclared(t *testing.T) {
	for _, mediaType := range []string{"text/html", "application/xhtml+xml"} {
		if _, err := documentextraction.DocumentFrom(
			t.Context(),
			[]byte(article),
			mediaType,
			canonicalurltest.CanonicalURLOf(t, "http://host.example/p"),
		); err != nil {
			t.Fatalf("DocumentFrom %s: %v", mediaType, err)
		}
	}
}

func TestExtractReportsNoLanguageWithoutATwoLetterLanguageTag(t *testing.T) {
	for _, opening := range []string{`<html>`, `<html lang="english">`, `<html lang="e1">`} {
		page := `<!DOCTYPE html>` + opening +
			`<head><title>Sample Article</title></head>` +
			`<body><article><p>` + longText + `</p></article></body></html>`

		doc, err := documentextraction.DocumentFrom(
			t.Context(),
			[]byte(page),
			"text/html",
			canonicalurltest.CanonicalURLOf(t, "http://host.example/p"),
		)
		if err != nil {
			t.Fatalf("DocumentFrom %s: %v", opening, err)
		}
		if doc.Language != "" {
			t.Errorf("%s yields language %q, want none", opening, doc.Language)
		}
	}
}

func TestLinkCountsResolveAgainstTheBaseHref(t *testing.T) {
	page := `<!DOCTYPE html><html><head><title>t</title>` +
		`<base href="http://other.example/base/"></head>` +
		`<body><p>` + longText + `</p><a href="sibling">s</a>` +
		`<a href="http://host.example/x">x</a></body></html>`

	doc, err := documentextraction.DocumentFrom(
		t.Context(),
		[]byte(page),
		"text/html",
		canonicalurltest.CanonicalURLOf(t, "http://host.example/dir/p"),
	)
	if err != nil {
		t.Fatalf("DocumentFrom: %v", err)
	}
	if doc.LocalLinks != 1 || doc.ExternalLinks != 1 {
		t.Fatalf("base href ignored, local=%d external=%d", doc.LocalLinks, doc.ExternalLinks)
	}
}

func TestAnUnresolvableBaseHrefLeavesThePageURLInPlace(t *testing.T) {
	page := `<!DOCTYPE html><html><head><title>t</title>` +
		`<base href="::not a url::"></head>` +
		`<body><p>` + longText + `</p><a href="/local">l</a></body></html>`

	doc, err := documentextraction.DocumentFrom(
		t.Context(),
		[]byte(page),
		"text/html",
		canonicalurltest.CanonicalURLOf(t, "http://host.example/dir/p"),
	)
	if err != nil {
		t.Fatalf("DocumentFrom: %v", err)
	}
	if doc.LocalLinks != 1 || doc.ExternalLinks != 0 {
		t.Fatalf("local=%d external=%d", doc.LocalLinks, doc.ExternalLinks)
	}
}

func TestARepeatedLinkCountsOnce(t *testing.T) {
	page := `<!DOCTYPE html><html><head><title>t</title></head>` +
		`<body><p>` + longText + `</p><a href="/one">a</a><a href="/one">b</a>` +
		`<a>no href</a><a href="::broken::">c</a></body></html>`

	doc, err := documentextraction.DocumentFrom(
		t.Context(),
		[]byte(page),
		"text/html",
		canonicalurltest.CanonicalURLOf(t, "http://host.example/dir/p"),
	)
	if err != nil {
		t.Fatalf("DocumentFrom: %v", err)
	}
	if doc.LocalLinks != 1 || doc.ExternalLinks != 0 {
		t.Fatalf("local=%d external=%d", doc.LocalLinks, doc.ExternalLinks)
	}
}
