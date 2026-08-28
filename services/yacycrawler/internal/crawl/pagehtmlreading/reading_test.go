package pagehtmlreading_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagerefusals"
)

const (
	pageLinkingNext      = `<html><body><a href="/next">next</a></body></html>`
	pageRefusingIndexing = `<html><head><meta name="robots" content="noindex">` +
		`</head><body><a href="/next">next</a></body></html>`
	pageRefusingLinkDiscovery = `<html><head><meta name="robots" content="nofollow">` +
		`</head><body><a href="/next">next</a></body></html>`
)

func pageHolding(t *testing.T, markup string) pagefetch.FetchedPage {
	t.Helper()
	return pagefetch.FetchedPage{
		LandedURL:   canonicalurltest.CanonicalURLOf(t, "http://host/"),
		ContentType: "text/html",
		Body:        []byte(markup),
	}
}

func readingOf(
	t *testing.T,
	page pagefetch.FetchedPage,
	ignored pagerefusals.IgnoredRefusals,
) pagehtmlreading.Reading {
	t.Helper()
	reading, err := pagehtmlreading.ReadingOfPage(t.Context(), page, ignored)
	if err != nil {
		t.Fatalf("ReadingOfPage: %v", err)
	}
	return reading
}

func TestReadingOfPageRejectsABodyThatIsNotHTML(t *testing.T) {
	page := pageHolding(t, pageLinkingNext)
	page.ContentType = "application/pdf"

	_, err := pagehtmlreading.ReadingOfPage(t.Context(), page, pagerefusals.IgnoredRefusals{})

	if !errors.Is(err, pagehtmlreading.ErrPageNotHTML) {
		t.Fatalf("want ErrPageNotHTML, got %v", err)
	}
}

func TestReadingOfPageReportsTheURLsThePageLinksTo(t *testing.T) {
	reading := readingOf(t, pageHolding(t, pageLinkingNext), pagerefusals.IgnoredRefusals{})

	if len(reading.DiscoveredURLs) != 1 ||
		reading.DiscoveredURLs[0] != canonicalurltest.CanonicalURLOf(t, "http://host/next") {
		t.Fatalf("want the linked url returned, got %v", reading.DiscoveredURLs)
	}
}

func TestReadingOfPageStillReportsLinksOnAPageThatRefusesIndexing(t *testing.T) {
	reading := readingOf(t, pageHolding(t, pageRefusingIndexing), pagerefusals.IgnoredRefusals{})

	if len(reading.DiscoveredURLs) != 1 {
		t.Fatalf("noindex leaves links followable, got %v", reading.DiscoveredURLs)
	}
}

func TestReadingOfPageReportsNoLinksWhenThePageRefusesLinkDiscovery(t *testing.T) {
	reading := readingOf(
		t, pageHolding(t, pageRefusingLinkDiscovery), pagerefusals.IgnoredRefusals{},
	)

	if len(reading.DiscoveredURLs) != 0 {
		t.Fatalf("nofollow suppresses linked urls, got %v", reading.DiscoveredURLs)
	}
}

func TestReadingOfPageHonorsARefusalStatedOutsideTheHTML(t *testing.T) {
	page := pageHolding(t, pageLinkingNext)
	page.RobotsDirectives = []string{"nofollow"}

	reading := readingOf(t, page, pagerefusals.IgnoredRefusals{})

	if len(reading.DiscoveredURLs) != 0 {
		t.Fatalf("nofollow suppresses linked urls, got %v", reading.DiscoveredURLs)
	}
}

func TestReadingOfPageDropsAnIndexingRefusalTheOrderIgnores(t *testing.T) {
	reading := readingOf(
		t,
		pageHolding(t, pageRefusingIndexing),
		pagerefusals.IgnoredRefusals{IndexingRefusal: true},
	)

	if reading.Refusals.RefusesIndexing {
		t.Fatal("an ignored indexing refusal is not reported")
	}
}
