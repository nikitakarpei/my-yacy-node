package scrapedpage_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/scrapedpage"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
)

func TestOfCarriesTheBytesTheReadReturned(t *testing.T) {
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/a")

	scraped := scrapedpage.Of(
		scraperequestcontract.ScrapeRequest{PageURL: pageURL, FetchURL: pageURL},
		pagefetch.FetchedPage{
			LandedURL:   pageURL,
			ContentType: "text/html",
			Body:        []byte("hello"),
		},
	)

	if scraped.ContentType != "text/html" || string(scraped.Body) != "hello" {
		t.Errorf("scraped page = %+v", scraped)
	}
}

func TestOfTakesALandingAsThePageWhenTheRequestReadsThePageAtItsOwnURL(t *testing.T) {
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/a")
	landedURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/b")

	scraped := scrapedpage.Of(
		scraperequestcontract.ScrapeRequest{PageURL: pageURL, FetchURL: pageURL},
		pagefetch.FetchedPage{LandedURL: landedURL},
	)

	if scraped.PageURL != landedURL {
		t.Errorf("page url = %q, want the landing %q", scraped.PageURL, landedURL)
	}
	if scraped.LandedURL != landedURL {
		t.Errorf("landed url = %q, want %q", scraped.LandedURL, landedURL)
	}
}

func TestOfKeepsThePageURLWhenTheRequestReadsThePageAtAnotherURL(t *testing.T) {
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/a")
	landedURL := canonicalurltest.CanonicalURLOf(t, "https://archive.example/b")

	scraped := scrapedpage.Of(
		scraperequestcontract.ScrapeRequest{
			PageURL:  pageURL,
			FetchURL: canonicalurltest.CanonicalURLOf(t, "https://archive.example/a"),
		},
		pagefetch.FetchedPage{LandedURL: landedURL},
	)

	if scraped.PageURL != pageURL {
		t.Errorf("page url = %q, want the named page url %q", scraped.PageURL, pageURL)
	}
	if scraped.LandedURL != landedURL {
		t.Errorf("landed url = %q, want %q", scraped.LandedURL, landedURL)
	}
}
