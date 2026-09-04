package pagehtml_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
)

func elementTreeOfHTML(t *testing.T, pageHTML string) pagehtml.ElementTree {
	t.Helper()
	tree, err := pagehtml.NewHTMLParser(&recordingMediaTypeObserver{}).ElementTreeFrom(
		t.Context(), "text/html", []byte(pageHTML),
	)
	if err != nil {
		t.Fatalf("ElementTreeFrom: %v", err)
	}
	return tree
}

func TestElementsNamedYieldsEveryElementOfThatNameInDocumentOrder(t *testing.T) {
	tree := elementTreeOfHTML(t, page)

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
	tree := elementTreeOfHTML(t, page)

	titles := 0
	for range tree.ElementsNamed("title") {
		titles++
	}

	if titles != 1 {
		t.Fatalf("title elements = %d, want 1", titles)
	}
}

func TestElementsNamedStopsWhenTheReaderStops(t *testing.T) {
	tree := elementTreeOfHTML(t, page)

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
	tree := elementTreeOfHTML(t, `<html><body><a HREF="/next">next</a></body></html>`)

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
