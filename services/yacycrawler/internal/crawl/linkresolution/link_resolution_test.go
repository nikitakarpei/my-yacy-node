package linkresolution_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkresolution"
)

const pageAddress = "http://host.example/dir/p"

func linksAgainst(
	baseURL canonicalurl.CanonicalURL,
	references ...string,
) []linkresolution.Link {
	links := make([]linkresolution.Link, 0, len(references))
	for _, reference := range references {
		links = append(links, linkresolution.Link{BaseURL: baseURL, Reference: reference})
	}
	return links
}

func TestURLsOfResolvesEveryLinkAgainstItsBaseURL(t *testing.T) {
	pageURL := canonicalurltest.CanonicalURLOf(t, pageAddress)
	otherBaseURL := canonicalurltest.CanonicalURLOf(t, "http://other.example/base/")
	links := append(
		linksAgainst(pageURL, "sibling", "/root"),
		linksAgainst(otherBaseURL, "child")...)

	urls := linkresolution.New(&recordingLinkResolutionObserver{}).
		URLsOf(t.Context(), pageURL, links)

	assertURLs(t, urls, []string{
		"http://host.example/dir/sibling",
		"http://host.example/root",
		"http://other.example/base/child",
	})
}

func TestURLsOfReturnsARepeatedLinkOnce(t *testing.T) {
	pageURL := canonicalurltest.CanonicalURLOf(t, pageAddress)

	urls := linkresolution.New(&recordingLinkResolutionObserver{}).URLsOf(
		t.Context(), pageURL, linksAgainst(pageURL, "/one", "/one", "http://host.example/one"),
	)

	assertURLs(t, urls, []string{"http://host.example/one"})
}

func TestURLsOfReportsTheLinksItCannotResolve(t *testing.T) {
	observer := &recordingLinkResolutionObserver{}
	pageURL := canonicalurltest.CanonicalURLOf(t, pageAddress)

	urls := linkresolution.New(observer).URLsOf(
		t.Context(), pageURL, linksAgainst(pageURL, "/one", "::broken::", "mailto:a@host.example"),
	)

	assertURLs(t, urls, []string{"http://host.example/one"})
	if observer.unresolvedLinks != 2 {
		t.Fatalf("unresolved links = %d, want 2", observer.unresolvedLinks)
	}
}

func TestURLsOfReportsNothingWhenEveryLinkResolves(t *testing.T) {
	observer := &recordingLinkResolutionObserver{}
	pageURL := canonicalurltest.CanonicalURLOf(t, pageAddress)

	linkresolution.New(observer).URLsOf(t.Context(), pageURL, linksAgainst(pageURL, "/one"))

	if observer.linkReports != 0 {
		t.Fatalf("link reports = %d, want 0", observer.linkReports)
	}
}

func TestBaseURLOfResolvesTheStatedBaseURLAgainstTheEnclosingOne(t *testing.T) {
	baseURL := linkresolution.New(&recordingLinkResolutionObserver{}).BaseURLOf(
		t.Context(), canonicalurltest.CanonicalURLOf(t, pageAddress), "../docs/",
	)

	if baseURL.String() != "http://host.example/docs/" {
		t.Fatalf("base url = %s, want http://host.example/docs/", baseURL)
	}
}

func TestBaseURLOfKeepsTheEnclosingBaseURLWhenNoneIsStated(t *testing.T) {
	enclosingBaseURL := canonicalurltest.CanonicalURLOf(t, pageAddress)

	baseURL := linkresolution.New(&recordingLinkResolutionObserver{}).BaseURLOf(
		t.Context(), enclosingBaseURL, "",
	)

	if baseURL != enclosingBaseURL {
		t.Fatalf("base url = %s, want %s", baseURL, enclosingBaseURL)
	}
}

func TestBaseURLOfKeepsTheEnclosingBaseURLWhenTheStatedOneCannotBeResolved(t *testing.T) {
	observer := &recordingLinkResolutionObserver{}
	enclosingBaseURL := canonicalurltest.CanonicalURLOf(t, pageAddress)

	baseURL := linkresolution.New(observer).
		BaseURLOf(t.Context(), enclosingBaseURL, "::not a url::")

	if baseURL != enclosingBaseURL {
		t.Fatalf("base url = %s, want %s", baseURL, enclosingBaseURL)
	}
	if observer.unresolvedBaseURLs != 1 {
		t.Fatalf("unresolved base urls = %d, want 1", observer.unresolvedBaseURLs)
	}
}

func TestEveryObserverHearsAnUnresolvedLink(t *testing.T) {
	first, second := &recordingLinkResolutionObserver{}, &recordingLinkResolutionObserver{}
	resolution := linkresolution.New(linkresolution.LinkResolutionObservers{first, second})
	pageURL := canonicalurltest.CanonicalURLOf(t, pageAddress)

	resolution.BaseURLOf(t.Context(), pageURL, "::not a url::")
	resolution.URLsOf(t.Context(), pageURL, linksAgainst(pageURL, "::broken::"))

	for _, observer := range []*recordingLinkResolutionObserver{first, second} {
		if observer.unresolvedBaseURLs != 1 || observer.unresolvedLinks != 1 {
			t.Fatalf("observer heard %+v, want one of each", *observer)
		}
	}
}

func assertURLs(t *testing.T, got []canonicalurl.CanonicalURL, want []string) {
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

type recordingLinkResolutionObserver struct {
	unresolvedBaseURLs int
	unresolvedLinks    int
	linkReports        int
}

func (o *recordingLinkResolutionObserver) BaseURLUnresolved(
	context.Context, canonicalurl.CanonicalURL, string, error,
) {
	o.unresolvedBaseURLs++
}

func (o *recordingLinkResolutionObserver) LinksUnresolved(
	_ context.Context, _ canonicalurl.CanonicalURL, links int,
) {
	o.linkReports++
	o.unresolvedLinks += links
}
