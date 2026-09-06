package pagehtmlreading_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkdiscovery"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
)

const (
	pageLinkingNext      = `<html><body><a href="/next">next</a></body></html>`
	pageRefusingIndexing = `<html><head><meta name="robots" content="noindex">` +
		`</head><body><a href="/next">next</a></body></html>`
	pageRefusingLinkDiscovery = `<html><head><meta name="robots" content="nofollow">` +
		`</head><body><a href="/next">next</a></body></html>`
)

func pageURL(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	return canonicalurltest.CanonicalURLOf(t, "http://host/")
}

func pageHolding(t *testing.T, markup string) pagefetch.FetchedPage {
	t.Helper()
	return pagefetch.FetchedPage{
		ContentType: "text/html",
		Body:        []byte(markup),
	}
}

func htmlPageReading() *pagehtmlreading.HTMLPageReading {
	return pagehtmlreading.NewHTMLPageReading(
		pagehtml.NewHTMLParser(silentMediaTypeObserver{}),
		linkdiscovery.NewLinkDiscovery(silentLinkResolutionObserver{}),
	)
}

func readingOf(t *testing.T, page pagefetch.FetchedPage) pagehtmlreading.Reading {
	t.Helper()
	reading, err := htmlPageReading().ReadingOfPage(t.Context(), pageURL(t), page)
	if err != nil {
		t.Fatalf("ReadingOfPage: %v", err)
	}
	return reading
}

func TestReadingOfPageRejectsABodyThatIsNotHTML(t *testing.T) {
	page := pageHolding(t, pageLinkingNext)
	page.ContentType = "application/pdf"

	_, err := htmlPageReading().ReadingOfPage(t.Context(), pageURL(t), page)

	if !errors.Is(err, pagehtmlreading.ErrPageNotHTML) {
		t.Fatalf("want ErrPageNotHTML, got %v", err)
	}
}

func TestReadingOfPageKeepsHTMLReadingFailuresDistinct(t *testing.T) {
	for name, readingError := range map[string]error{
		"charset": pagehtml.ErrCharsetUnreadable,
		"parse":   pagehtml.ErrHTMLUnparseable,
	} {
		t.Run(name, func(t *testing.T) {
			reading := pagehtmlreading.NewHTMLPageReading(
				failingHTMLParser{err: readingError},
				linkDiscoveryWithoutLinks{},
			)

			_, err := reading.ReadingOfPage(
				t.Context(),
				pageURL(t),
				pageHolding(t, pageLinkingNext),
			)

			if !errors.Is(err, readingError) {
				t.Fatalf("want %v, got %v", readingError, err)
			}
			if errors.Is(err, pagehtmlreading.ErrPageNotHTML) {
				t.Fatalf("HTML reading error flattened into ErrPageNotHTML: %v", err)
			}
		})
	}
}

func TestReadingOfPageReportsTheURLsThePageLinksTo(t *testing.T) {
	reading := readingOf(t, pageHolding(t, pageLinkingNext))

	if len(reading.DiscoveredURLs) != 1 ||
		reading.DiscoveredURLs[0] != canonicalurltest.CanonicalURLOf(t, "http://host/next") {
		t.Fatalf("want the linked url returned, got %v", reading.DiscoveredURLs)
	}
}

func TestReadingOfPageStillReportsLinksOnAPageThatRefusesIndexing(t *testing.T) {
	reading := readingOf(t, pageHolding(t, pageRefusingIndexing))

	if len(reading.DiscoveredURLs) != 1 {
		t.Fatalf("noindex leaves links followable, got %v", reading.DiscoveredURLs)
	}
}

func TestReadingOfPageReportsNoLinksWhenThePageRefusesLinkDiscovery(t *testing.T) {
	reading := readingOf(
		t, pageHolding(t, pageRefusingLinkDiscovery),
	)

	if len(reading.DiscoveredURLs) != 0 {
		t.Fatalf("nofollow suppresses linked urls, got %v", reading.DiscoveredURLs)
	}
}

func TestReadingOfPageHonorsARefusalStatedOutsideTheHTML(t *testing.T) {
	page := pageHolding(t, pageLinkingNext)
	page.RobotsDirectives = []string{"nofollow"}

	reading := readingOf(t, page)

	if len(reading.DiscoveredURLs) != 0 {
		t.Fatalf("nofollow suppresses linked urls, got %v", reading.DiscoveredURLs)
	}
}

type silentMediaTypeObserver struct{}

func (silentMediaTypeObserver) MediaTypeUnparsed(context.Context, string, error) {}

type failingHTMLParser struct{ err error }

func (parser failingHTMLParser) ElementTreeFrom(
	context.Context,
	string,
	[]byte,
) (pagehtml.ElementTree, error) {
	return pagehtml.ElementTree{}, parser.err
}

type linkDiscoveryWithoutLinks struct{}

func (linkDiscoveryWithoutLinks) LinkedURLsFrom(
	context.Context,
	pagehtml.ElementTree,
	canonicalurl.CanonicalURL,
) []canonicalurl.CanonicalURL {
	return nil
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
