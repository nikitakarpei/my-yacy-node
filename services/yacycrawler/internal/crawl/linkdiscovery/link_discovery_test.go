package linkdiscovery_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkdiscovery"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
)

func linkedURLsFrom(
	t *testing.T,
	pageURL, pageHTML string,
) ([]canonicalurl.CanonicalURL, *recordingLinkResolutionObserver) {
	t.Helper()
	observer := &recordingLinkResolutionObserver{}
	elementTree, err := pagehtml.NewHTMLParser(silentMediaTypeObserver{}).ElementTreeFrom(
		t.Context(), "text/html", []byte(pageHTML),
	)
	if err != nil {
		t.Fatalf("ElementTreeFrom: %v", err)
	}
	return linkdiscovery.NewLinkDiscovery(observer).LinkedURLsFrom(
		t.Context(), elementTree, canonicalurltest.CanonicalURLOf(t, pageURL),
	), observer
}

func TestRelativeLinksResolveAgainstThePageURL(t *testing.T) {
	urls, _ := linkedURLsFrom(t, "http://host.example/dir/p",
		`<html><body><a href="sibling">s</a><a href="/root">r</a></body></html>`,
	)

	want := []string{"http://host.example/dir/sibling", "http://host.example/root"}
	assertURLs(t, urls, want)
}

func TestRelativeLinksResolveAgainstTheBaseHref(t *testing.T) {
	urls, _ := linkedURLsFrom(t, "http://host.example/dir/p",
		`<html><head><base href="http://other.example/base/"></head>`+
			`<body><a href="sibling">s</a></body></html>`,
	)

	assertURLs(t, urls, []string{"http://other.example/base/sibling"})
}

func TestAnUnresolvableBaseHrefLeavesThePageURLInPlace(t *testing.T) {
	urls, observer := linkedURLsFrom(t, "http://host.example/dir/p",
		`<html><head><base href="::not a url::"></head>`+
			`<body><a href="sibling">s</a></body></html>`,
	)

	assertURLs(t, urls, []string{"http://host.example/dir/sibling"})
	if observer.unresolvedBaseHrefs != 1 {
		t.Fatalf("unresolved base hrefs = %d, want 1", observer.unresolvedBaseHrefs)
	}
}

func TestARepeatedLinkIsDiscoveredOnce(t *testing.T) {
	urls, observer := linkedURLsFrom(t, "http://host.example/dir/p",
		`<html><body><a href="/one">a</a><a href="/one">b</a>`+
			`<a>no href</a><a href="::broken::">c</a></body></html>`,
	)

	assertURLs(t, urls, []string{"http://host.example/one"})
	if observer.unresolvedLinkHrefs != 1 {
		t.Fatalf("unresolved link hrefs = %d, want 1", observer.unresolvedLinkHrefs)
	}
}

func TestAPageWithoutLinksDiscoversNothing(t *testing.T) {
	urls, observer := linkedURLsFrom(
		t, "http://host.example/", `<html><body>text</body></html>`,
	)

	if len(urls) != 0 {
		t.Fatalf("want no links, got %v", urls)
	}
	if observer.unresolvedLinkHrefs != 0 {
		t.Fatalf("unresolved link hrefs = %d, want 0", observer.unresolvedLinkHrefs)
	}
}

func assertURLs(t *testing.T, got []canonicalurl.CanonicalURL, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("links = %v, want %v", got, want)
	}
	for i, wanted := range want {
		if got[i].String() != wanted {
			t.Fatalf("links = %v, want %v", got, want)
		}
	}
}

type silentMediaTypeObserver struct{}

func (silentMediaTypeObserver) MediaTypeUnparsed(context.Context, string, error) {}

type recordingLinkResolutionObserver struct {
	unresolvedBaseHrefs int
	unresolvedLinkHrefs int
}

func (o *recordingLinkResolutionObserver) BaseHrefUnresolved(
	context.Context, canonicalurl.CanonicalURL, string, error,
) {
	o.unresolvedBaseHrefs++
}

func (o *recordingLinkResolutionObserver) LinkHrefsUnresolved(
	_ context.Context, _ canonicalurl.CanonicalURL, hrefs int,
) {
	o.unresolvedLinkHrefs += hrefs
}
