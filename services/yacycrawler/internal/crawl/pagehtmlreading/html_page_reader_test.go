package pagehtmlreading_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkresolution"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagereading"
)

const (
	pageAddress          = "http://host.example/dir/p"
	pageLinkingNext      = `<html><body><a href="/next">next</a></body></html>`
	pageRefusingIndexing = `<html><head><meta name="robots" content="noindex">` +
		`</head><body><a href="/next">next</a></body></html>`
	pageRefusingLinkDiscovery = `<html><head><meta name="robots" content="nofollow">` +
		`</head><body><a href="/next">next</a></body></html>`
)

func contentTypeOf(t *testing.T, declared string) pagereading.ContentType {
	t.Helper()
	contentType, err := pagereading.ContentTypeOf(declared)
	if err != nil {
		t.Fatalf("ContentTypeOf %q: %v", declared, err)
	}
	return contentType
}

func readingOfBody(
	t *testing.T,
	declaredContentType, markup string,
) (pagereading.BodyReading, bool) {
	t.Helper()
	reading, inPageFormat, err := pagehtmlreading.NewHTMLPageReader(
		linkresolution.New(silentLinkResolutionObserver{}),
	).ReadingOfBody(
		t.Context(),
		canonicalurltest.CanonicalURLOf(t, pageAddress),
		contentTypeOf(t, declaredContentType),
		[]byte(markup),
	)
	if err != nil {
		t.Fatalf("ReadingOfBody: %v", err)
	}
	return reading, inPageFormat
}

func readingOfHTML(t *testing.T, markup string) pagereading.BodyReading {
	t.Helper()
	reading, inPageFormat := readingOfBody(t, "text/html", markup)
	if !inPageFormat {
		t.Fatal("want an HTML body read as HTML")
	}
	return reading
}

func TestReadingOfBodyLeavesABodyThatIsNotHTMLUnread(t *testing.T) {
	for _, declaredContentType := range []string{"application/pdf", "text/xml", "application/rss+xml"} {
		if _, inPageFormat := readingOfBody(t, declaredContentType, pageLinkingNext); inPageFormat {
			t.Errorf("%s read as HTML", declaredContentType)
		}
	}
}

func TestReadingOfBodyReadsTheHTMLMediaTypes(t *testing.T) {
	for _, declaredContentType := range []string{
		"text/html",
		"text/html; charset=utf-8",
		"application/xhtml+xml",
	} {
		reading, inPageFormat := readingOfBody(t, declaredContentType, pageLinkingNext)
		if !inPageFormat || len(reading.LinkedURLs) != 1 {
			t.Errorf(
				"%s: read = %v, links = %v",
				declaredContentType,
				inPageFormat,
				reading.LinkedURLs,
			)
		}
	}
}

func TestReadingOfBodyFailsOnAnHTMLBodyWhoseCharsetCannotBeRead(t *testing.T) {
	_, inPageFormat, err := pagehtmlreading.NewHTMLPageReader(
		linkresolution.New(silentLinkResolutionObserver{}),
	).ReadingOfBody(
		t.Context(),
		canonicalurltest.CanonicalURLOf(t, pageAddress), contentTypeOf(t, "text/html"), nil,
	)

	if err == nil || !inPageFormat {
		t.Fatalf("want an HTML reading failure, got read = %v, err = %v", inPageFormat, err)
	}
}

func TestReadingOfBodyReportsTheURLsTheAnchorsLinkToInDocumentOrder(t *testing.T) {
	reading := readingOfHTML(t,
		`<html><body><a href="sibling">s</a><a HREF="/root">r</a><a>no href</a>`+
			`<link href="/style.css"></body></html>`,
	)

	assertURLs(t, reading.LinkedURLs, "http://host.example/dir/sibling", "http://host.example/root")
}

func TestReadingOfBodyResolvesTheAnchorsAgainstTheFirstBaseHref(t *testing.T) {
	reading := readingOfHTML(t,
		`<html><head><base target="_blank"><base href="http://other.example/base/">`+
			`<base href="http://third.example/"></head>`+
			`<body><a href="sibling">s</a></body></html>`,
	)

	assertURLs(t, reading.LinkedURLs, "http://other.example/base/sibling")
}

func TestReadingOfBodyReportsTheIndexingRefusalOfTheRobotsMetaTag(t *testing.T) {
	reading := readingOfHTML(t, pageRefusingIndexing)

	if !reading.Refusals.RefusesIndexing {
		t.Fatalf("want the meta noindex read, got %+v", reading.Refusals)
	}
}

func TestReadingOfBodyReportsTheLinksOfAPageThatRefusesLinkDiscovery(t *testing.T) {
	reading := readingOfHTML(t, pageRefusingLinkDiscovery)

	if !reading.Refusals.RefusesLinkDiscovery {
		t.Fatalf("want the meta nofollow read, got %+v", reading.Refusals)
	}
	if len(reading.LinkedURLs) != 1 {
		t.Fatalf("want the links the markup holds, got %v", reading.LinkedURLs)
	}
}

func assertURLs(t *testing.T, got []canonicalurl.CanonicalURL, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("urls = %v, want %v", got, want)
	}
	for i, wanted := range want {
		if got[i].String() != wanted {
			t.Fatalf("urls = %v, want %v", got, want)
		}
	}
}

type silentLinkResolutionObserver struct{}

func (silentLinkResolutionObserver) BaseURLUnresolved(
	context.Context, canonicalurl.CanonicalURL, string, error,
) {
}

func (silentLinkResolutionObserver) LinksUnresolved(
	context.Context, canonicalurl.CanonicalURL, int,
) {
}
