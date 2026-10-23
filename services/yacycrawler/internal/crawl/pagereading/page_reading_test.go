package pagereading_test

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
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagereading"
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

func pageNotHTML(t *testing.T) pagefetch.FetchedPage {
	t.Helper()
	page := pageHolding(t, pageLinkingNext)
	page.ContentType = "application/pdf"
	return page
}

type countingRefusalEnforcementObserver struct {
	linkDiscoveryRefusalsEnforced int
}

func (o *countingRefusalEnforcementObserver) LinkDiscoveryRefusalEnforced(
	context.Context,
	canonicalurl.CanonicalURL,
) {
	o.linkDiscoveryRefusalsEnforced++
}

func pageReading() *pagereading.PageReading {
	return pageReadingObservedBy(&countingRefusalEnforcementObserver{})
}

func pageReadingObservedBy(
	refusalEnforcementObserver pagereading.RefusalEnforcementObserver,
) *pagereading.PageReading {
	return pagereading.NewPageReading(
		pagehtmlreading.NewHTMLPageReading(
			pagehtml.NewHTMLParser(silentMediaTypeObserver{}),
			linkdiscovery.NewLinkDiscovery(silentLinkResolutionObserver{}),
		),
		refusalEnforcementObserver,
	)
}

func readingOf(t *testing.T, page pagefetch.FetchedPage) pagereading.Reading {
	t.Helper()
	reading, err := pageReading().ReadingOfPage(t.Context(), pageURL(t), page)
	if err != nil {
		t.Fatalf("ReadingOfPage: %v", err)
	}
	return reading
}

func TestReadingOfPageReportsNoLinksOnAPageThatIsNotHTML(t *testing.T) {
	reading := readingOf(t, pageNotHTML(t))

	if len(reading.DiscoveredURLs) != 0 {
		t.Fatalf("a page that is not HTML links nowhere, got %v", reading.DiscoveredURLs)
	}
}

func TestReadingOfPageHonorsAnIndexingRefusalInTheHeadersOfAPageThatIsNotHTML(t *testing.T) {
	page := pageNotHTML(t)
	page.RobotsTagValues = []string{"noindex"}

	reading := readingOf(t, page)

	if !reading.Refusals.RefusesIndexing {
		t.Fatalf("want the header noindex read, got %+v", reading.Refusals)
	}
}

func TestReadingOfPageKeepsAnHTMLReadingFailure(t *testing.T) {
	htmlReadingFailure := errors.New("html unreadable")
	reading := pagereading.NewPageReading(
		failingHTMLPageReading{err: htmlReadingFailure},
		&countingRefusalEnforcementObserver{},
	)

	_, err := reading.ReadingOfPage(t.Context(), pageURL(t), pageHolding(t, pageLinkingNext))

	if !errors.Is(err, htmlReadingFailure) {
		t.Fatalf("want %v, got %v", htmlReadingFailure, err)
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

	if !reading.Refusals.RefusesIndexing {
		t.Fatalf("want the meta noindex kept, got %+v", reading.Refusals)
	}
	if len(reading.DiscoveredURLs) != 1 {
		t.Fatalf("noindex leaves links followable, got %v", reading.DiscoveredURLs)
	}
}

func TestReadingOfPageJoinsTheRefusalsOfTheHeadersAndTheHTML(t *testing.T) {
	page := pageHolding(t, pageRefusingIndexing)
	page.RobotsTagValues = []string{"nofollow"}

	reading := readingOf(t, page)

	if !reading.Refusals.RefusesIndexing || !reading.Refusals.RefusesLinkDiscovery {
		t.Fatalf("want both refusals, got %+v", reading.Refusals)
	}
}

func TestReadingOfPageReportsNoLinksWhenThePageRefusesLinkDiscovery(t *testing.T) {
	headerRefusal := pageHolding(t, pageLinkingNext)
	headerRefusal.RobotsTagValues = []string{"nofollow"}
	for name, page := range map[string]pagefetch.FetchedPage{
		"html":   pageHolding(t, pageRefusingLinkDiscovery),
		"header": headerRefusal,
	} {
		t.Run(name, func(t *testing.T) {
			reading := readingOf(t, page)

			if len(reading.DiscoveredURLs) != 0 {
				t.Fatalf("nofollow suppresses linked urls, got %v", reading.DiscoveredURLs)
			}
		})
	}
}

func TestReadingOfPageReportsTheLinkDiscoveryRefusalItEnforced(t *testing.T) {
	observer := &countingRefusalEnforcementObserver{}

	_, err := pageReadingObservedBy(observer).ReadingOfPage(
		t.Context(), pageURL(t), pageHolding(t, pageRefusingLinkDiscovery),
	)
	if err != nil {
		t.Fatalf("ReadingOfPage: %v", err)
	}

	if observer.linkDiscoveryRefusalsEnforced != 1 {
		t.Fatalf(
			"link discovery refusals enforced = %d, want 1",
			observer.linkDiscoveryRefusalsEnforced,
		)
	}
}

type failingHTMLPageReading struct{ err error }

func (reading failingHTMLPageReading) ReadingOfPage(
	context.Context,
	canonicalurl.CanonicalURL,
	pagefetch.FetchedPage,
) (pagehtmlreading.Reading, error) {
	return pagehtmlreading.Reading{}, reading.err
}

type silentMediaTypeObserver struct{}

func (silentMediaTypeObserver) MediaTypeUnparsed(context.Context, string, error) {}

type silentLinkResolutionObserver struct{}

func (silentLinkResolutionObserver) BaseURLUnresolved(
	context.Context, canonicalurl.CanonicalURL, string, error,
) {
}

func (silentLinkResolutionObserver) LinksUnresolved(
	context.Context, canonicalurl.CanonicalURL, int,
) {
}
