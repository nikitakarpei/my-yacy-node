package pagemarkup_test

import (
	"errors"
	"testing"

	"golang.org/x/net/html/atom"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagemarkup"
)

const page = `<html><head><title>t</title></head><body><a href="/next">next</a></body></html>`

func TestMarkupFromRejectsABodyThatIsNotHTML(t *testing.T) {
	if _, err := pagemarkup.MarkupFrom(t.Context(), "application/pdf", []byte(page)); !errors.Is(
		err, pagemarkup.ErrNotHTML,
	) {
		t.Fatalf("want ErrNotHTML, got %v", err)
	}
}

func TestMarkupFromAcceptsTheHTMLMediaTypes(t *testing.T) {
	for _, contentType := range []string{
		"text/html",
		"text/html; charset=utf-8",
		"application/xhtml+xml",
		"text/html;",
	} {
		if _, err := pagemarkup.MarkupFrom(t.Context(), contentType, []byte(page)); err != nil {
			t.Errorf("MarkupFrom %q: %v", contentType, err)
		}
	}
}

func TestElementsYieldsEveryElementInDocumentOrder(t *testing.T) {
	markup, err := pagemarkup.MarkupFrom(t.Context(), "text/html", []byte(page))
	if err != nil {
		t.Fatalf("MarkupFrom: %v", err)
	}

	var seen []atom.Atom
	for node := range markup.Elements() {
		seen = append(seen, node.DataAtom)
	}

	want := []atom.Atom{atom.Html, atom.Head, atom.Title, atom.Body, atom.A}
	if len(seen) != len(want) {
		t.Fatalf("elements = %v, want %v", seen, want)
	}
	for i, atomWanted := range want {
		if seen[i] != atomWanted {
			t.Fatalf("elements = %v, want %v", seen, want)
		}
	}
}

func TestElementsStopsWhenTheReaderStops(t *testing.T) {
	markup, err := pagemarkup.MarkupFrom(t.Context(), "text/html", []byte(page))
	if err != nil {
		t.Fatalf("MarkupFrom: %v", err)
	}

	visited := 0
	for range markup.Elements() {
		visited++
		break
	}

	if visited != 1 {
		t.Fatalf("want the walk stopped after one element, visited %d", visited)
	}
}

func TestAttributeOfReadsAKeyWhateverItsCase(t *testing.T) {
	markup, err := pagemarkup.MarkupFrom(t.Context(), "text/html", []byte(
		`<html><body><a HREF="/next">next</a></body></html>`,
	))
	if err != nil {
		t.Fatalf("MarkupFrom: %v", err)
	}

	for node := range markup.Elements() {
		if node.DataAtom != atom.A {
			continue
		}
		href, ok := pagemarkup.AttributeOf(node, "href")
		if !ok || href != "/next" {
			t.Fatalf("href = %q, present = %v", href, ok)
		}
		if _, ok := pagemarkup.AttributeOf(node, "rel"); ok {
			t.Fatal("an absent attribute reports itself absent")
		}
	}
}
