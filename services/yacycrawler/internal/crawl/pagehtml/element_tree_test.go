package pagehtml_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
)

const page = `<html><head><title>t</title></head><body><a href="/first">first</a><a href="/second">second</a></body></html>`

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

func TestElementsNamedYieldsEveryElementOfThatNameInDocumentOrder(t *testing.T) {
	tree, err := pagehtml.ElementTreeFrom(t.Context(), "text/html", []byte(page))
	if err != nil {
		t.Fatalf("ElementTreeFrom: %v", err)
	}

	var hrefs []string
	for element := range tree.ElementsNamed("a") {
		href, _ := element.AttributeOf("href")
		hrefs = append(hrefs, href)
	}

	want := []string{"/first", "/second"}
	if len(hrefs) != len(want) {
		t.Fatalf("hrefs = %v, want %v", hrefs, want)
	}
	for i, hrefWanted := range want {
		if hrefs[i] != hrefWanted {
			t.Fatalf("hrefs = %v, want %v", hrefs, want)
		}
	}
}

func TestElementsNamedSkipsEveryOtherElement(t *testing.T) {
	tree, err := pagehtml.ElementTreeFrom(t.Context(), "text/html", []byte(page))
	if err != nil {
		t.Fatalf("ElementTreeFrom: %v", err)
	}

	titles := 0
	for range tree.ElementsNamed("title") {
		titles++
	}

	if titles != 1 {
		t.Fatalf("title elements = %d, want 1", titles)
	}
}

func TestElementsNamedStopsWhenTheReaderStops(t *testing.T) {
	tree, err := pagehtml.ElementTreeFrom(t.Context(), "text/html", []byte(page))
	if err != nil {
		t.Fatalf("ElementTreeFrom: %v", err)
	}

	visited := 0
	for range tree.ElementsNamed("a") {
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

	for element := range tree.ElementsNamed("a") {
		href, ok := element.AttributeOf("href")
		if !ok || href != "/next" {
			t.Fatalf("href = %q, present = %v", href, ok)
		}
		if _, ok := element.AttributeOf("rel"); ok {
			t.Fatal("an absent attribute reports itself absent")
		}
	}
}
