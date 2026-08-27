package pagehtml_test

import (
	"errors"
	"testing"

	"golang.org/x/net/html/atom"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
)

const page = `<html><head><title>t</title></head><body><a href="/next">next</a></body></html>`

func TestElementTreeFromRejectsABodyThatIsNotHTML(t *testing.T) {
	if _, err := pagehtml.ElementTreeFrom(t.Context(), "application/pdf", []byte(page)); !errors.Is(
		err, pagehtml.ErrNotHTML,
	) {
		t.Fatalf("want ErrNotHTML, got %v", err)
	}
}

func TestElementTreeFromAcceptsTheHTMLMediaTypes(t *testing.T) {
	for _, contentType := range []string{
		"text/html",
		"text/html; charset=utf-8",
		"application/xhtml+xml",
		"text/html;",
	} {
		if _, err := pagehtml.ElementTreeFrom(t.Context(), contentType, []byte(page)); err != nil {
			t.Errorf("ElementTreeFrom %q: %v", contentType, err)
		}
	}
}

func TestElementsYieldsEveryElementInDocumentOrder(t *testing.T) {
	tree, err := pagehtml.ElementTreeFrom(t.Context(), "text/html", []byte(page))
	if err != nil {
		t.Fatalf("ElementTreeFrom: %v", err)
	}

	var seen []atom.Atom
	for node := range tree.Elements() {
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
	tree, err := pagehtml.ElementTreeFrom(t.Context(), "text/html", []byte(page))
	if err != nil {
		t.Fatalf("ElementTreeFrom: %v", err)
	}

	visited := 0
	for range tree.Elements() {
		visited++
		break
	}

	if visited != 1 {
		t.Fatalf("want the walk stopped after one element, visited %d", visited)
	}
}

func TestAttributeOfReadsAKeyWhateverItsCase(t *testing.T) {
	tree, err := pagehtml.ElementTreeFrom(t.Context(), "text/html", []byte(
		`<html><body><a HREF="/next">next</a></body></html>`,
	))
	if err != nil {
		t.Fatalf("ElementTreeFrom: %v", err)
	}

	for node := range tree.Elements() {
		if node.DataAtom != atom.A {
			continue
		}
		href, ok := pagehtml.AttributeOf(node, "href")
		if !ok || href != "/next" {
			t.Fatalf("href = %q, present = %v", href, ok)
		}
		if _, ok := pagehtml.AttributeOf(node, "rel"); ok {
			t.Fatal("an absent attribute reports itself absent")
		}
	}
}
