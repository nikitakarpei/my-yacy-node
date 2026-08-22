package html_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/mediaextractors/html"
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
	doc, err := html.New().
		Extract(t.Context(), "http://host.example/dir/p", "text/html", []byte(article))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if doc.Title != "Sample Article" {
		t.Fatalf("title = %q", doc.Title)
	}
	if doc.Format != contentformatgraph.FormatDocumentHTML {
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

func TestExtractHonorsMetaRobots(t *testing.T) {
	page := `<!DOCTYPE html><html lang="en"><head><title>T</title>
<meta name="robots" content="noindex,nofollow"></head>
<body><article><p>` + longText + `</p><p>` + longText + `</p></article></body></html>`
	doc, err := html.New().
		Extract(t.Context(), "http://host.example/p", "text/html", []byte(page))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if !doc.RefusesIndexing || !doc.RefusesLinkDiscovery {
		t.Fatalf("meta robots not honored: %+v", doc)
	}
}

func TestExtractYieldsWholeDocument(t *testing.T) {
	doc, err := html.New().
		Extract(t.Context(), "http://host.example/dir/p", "text/html", []byte(article))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	body := string(doc.Body)
	if !strings.Contains(body, "<title>Sample Article</title>") {
		t.Fatalf("whole document should retain head markup: %q", body)
	}
}

func TestMediaTypesDeclared(t *testing.T) {
	got := html.New().MediaTypes()
	if len(got) != 2 || got[0] != "text/html" {
		t.Fatalf("unexpected media types: %v", got)
	}
}

func TestExtractReportsNoLanguageWithoutATwoLetterLanguageTag(t *testing.T) {
	for _, opening := range []string{`<html>`, `<html lang="english">`, `<html lang="e1">`} {
		page := `<!DOCTYPE html>` + opening +
			`<head><title>Sample Article</title></head>` +
			`<body><article><p>` + longText + `</p></article></body></html>`

		doc, err := html.New().
			Extract(t.Context(), "http://host.example/p", "text/html", []byte(page))
		if err != nil {
			t.Fatalf("Extract %s: %v", opening, err)
		}
		if doc.Language != "" {
			t.Errorf("%s yields language %q, want none", opening, doc.Language)
		}
	}
}

func TestExtractResolvesLinksAgainstTheBaseHref(t *testing.T) {
	page := `<!DOCTYPE html><html lang="en"><head><title>T</title>
<base href="http://other.example/dir/"></head>
<body><a href="page">relative</a></body></html>`

	doc, err := html.New().
		Extract(t.Context(), "http://host.example/p", "text/html", []byte(page))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(doc.DiscoveredURLs) != 1 ||
		doc.DiscoveredURLs[0].String() != "http://other.example/dir/page" {
		t.Fatalf("discovered = %v", doc.DiscoveredURLs)
	}
	if doc.LocalLinks != 1 || doc.ExternalLinks != 0 {
		t.Fatalf("links local=%d external=%d", doc.LocalLinks, doc.ExternalLinks)
	}
}

func TestExtractDiscoversOnlyAbsoluteLinksOfAPageWithoutACanonicalURL(t *testing.T) {
	page := `<!DOCTYPE html><html lang="en"><head><title>T</title></head>
<body><a href="/relative">relative</a>
<a href="http://other.example/ext">absolute</a></body></html>`

	doc, err := html.New().Extract(t.Context(), "file:///tmp/p", "text/html", []byte(page))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(doc.DiscoveredURLs) != 1 ||
		doc.DiscoveredURLs[0].String() != "http://other.example/ext" {
		t.Fatalf("discovered = %v", doc.DiscoveredURLs)
	}
}
