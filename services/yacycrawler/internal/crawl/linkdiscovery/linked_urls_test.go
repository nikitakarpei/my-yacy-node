package linkdiscovery_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkdiscovery"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagemarkup"
)

func linkedURLsFrom(t *testing.T, pageURL, markup string) []canonicalurl.CanonicalURL {
	t.Helper()
	parsed, err := pagemarkup.MarkupFrom(t.Context(), "text/html", []byte(markup))
	if err != nil {
		t.Fatalf("MarkupFrom: %v", err)
	}
	return linkdiscovery.LinkedURLsFrom(
		t.Context(), parsed, canonicalurltest.CanonicalURLOf(t, pageURL),
	)
}

func TestRelativeLinksResolveAgainstThePageURL(t *testing.T) {
	urls := linkedURLsFrom(t, "http://host.example/dir/p",
		`<html><body><a href="sibling">s</a><a href="/root">r</a></body></html>`,
	)

	want := []string{"http://host.example/dir/sibling", "http://host.example/root"}
	assertURLs(t, urls, want)
}

func TestRelativeLinksResolveAgainstTheBaseHref(t *testing.T) {
	urls := linkedURLsFrom(t, "http://host.example/dir/p",
		`<html><head><base href="http://other.example/base/"></head>`+
			`<body><a href="sibling">s</a></body></html>`,
	)

	assertURLs(t, urls, []string{"http://other.example/base/sibling"})
}

func TestAnUnresolvableBaseHrefLeavesThePageURLInPlace(t *testing.T) {
	urls := linkedURLsFrom(t, "http://host.example/dir/p",
		`<html><head><base href="::not a url::"></head>`+
			`<body><a href="sibling">s</a></body></html>`,
	)

	assertURLs(t, urls, []string{"http://host.example/dir/sibling"})
}

func TestARepeatedLinkIsDiscoveredOnce(t *testing.T) {
	urls := linkedURLsFrom(t, "http://host.example/dir/p",
		`<html><body><a href="/one">a</a><a href="/one">b</a>`+
			`<a>no href</a><a href="::broken::">c</a></body></html>`,
	)

	assertURLs(t, urls, []string{"http://host.example/one"})
}

func TestAPageWithoutLinksDiscoversNothing(t *testing.T) {
	if urls := linkedURLsFrom(t, "http://host.example/", `<html><body>text</body></html>`); len(
		urls,
	) != 0 {
		t.Fatalf("want no links, got %v", urls)
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
