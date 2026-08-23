package pagescrape_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
)

const originHTML = `<html lang="en"><head><title>Hi</title></head>` +
	`<body><p>words here</p><a href="/next">next</a>` +
	`<a href="http://elsewhere.example/x">away</a></body></html>`

type fakeFetch struct {
	outcome pagefetch.FetchOutcome
	err     error
}

func (f fakeFetch) Fetch(
	context.Context,
	canonicalurl.CanonicalURL,
	pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	return f.outcome, f.err
}

func succeededWith(t *testing.T, finalURL, contentType, body string) pagefetch.FetchOutcome {
	t.Helper()
	return pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			FinalURL:    canonicalurltest.CanonicalURLOf(t, finalURL),
			ContentType: contentType,
			Body:        []byte(body),
		},
	}
}

func newScraper(t *testing.T, fetch pagefetch.Fetcher) *pagescrape.Scraper {
	t.Helper()
	scraper, err := pagescrape.New(fetch)
	if err != nil {
		t.Fatalf("new scraper: %v", err)
	}
	return scraper
}

func scrape(t *testing.T, outcome pagefetch.FetchOutcome) (pagescrape.ScrapedPage, bool) {
	t.Helper()
	page, scraped, err := newScraper(t, fakeFetch{outcome: outcome}).
		Scrape(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "http://host/"),
			documentextraction.FormatMarkdown,
		)
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}
	return page, scraped
}

func TestScrapeDerivesTheTargetFormatOfTheFetchedDocument(t *testing.T) {
	page, scraped := scrape(t, succeededWith(t, "http://HOST:80/a", "text/html", originHTML))

	if !scraped {
		t.Fatal("want a scraped page")
	}
	if page.CanonicalURL.String() != "http://host/a" {
		t.Errorf("canonical url = %q, want http://host/a", page.CanonicalURL)
	}
	if !strings.Contains(string(page.Content), "words here") {
		t.Errorf("content = %q, want it to carry the page text", page.Content)
	}
}

func TestScrapeCarriesWhatTheDocumentSaysAboutItself(t *testing.T) {
	page, scraped := scrape(t, succeededWith(t, "http://host/a", "text/html", originHTML))

	if !scraped {
		t.Fatal("want a scraped page")
	}
	if page.Title != "Hi" {
		t.Errorf("title = %q, want Hi", page.Title)
	}
	if page.Language != "en" {
		t.Errorf("language = %q, want en", page.Language)
	}
	if page.LocalLinks != 1 || page.ExternalLinks != 1 {
		t.Errorf("links = %d local, %d external, want 1 and 1",
			page.LocalLinks, page.ExternalLinks)
	}
}

func TestScrapeYieldsNoPageForAnUnreadableMediaType(t *testing.T) {
	page, scraped := scrape(t, succeededWith(t,
		"http://host/a.bin", "application/octet-stream", "\x00\x01",
	))

	if scraped {
		t.Fatalf("want no page, got %+v", page)
	}
}

func TestScrapeYieldsNoPageWhenTheFetchHoldsNoContent(t *testing.T) {
	for _, status := range []pagefetch.FetchStatus{
		pagefetch.FetchNotModified,
		pagefetch.FetchCeased,
		pagefetch.FetchNotAPage,
	} {
		if page, scraped := scrape(t, pagefetch.FetchOutcome{Status: status}); scraped {
			t.Errorf("status %d yielded %+v, want no page", status, page)
		}
	}
}

func TestScrapeFailsWhenAnotherAttemptCanSucceed(t *testing.T) {
	attempts := map[string]fakeFetch{
		"transport error": {err: errors.New("dial broke")},
		"fetch failed":    {outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchFailed}},
		"fetch deferred": {outcome: pagefetch.FetchOutcome{
			Status:   pagefetch.FetchDeferred,
			DeferFor: time.Second,
		}},
	}
	for name, fetch := range attempts {
		_, _, err := newScraper(t, fetch).Scrape(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "http://host/"),
			documentextraction.FormatMarkdown,
		)
		if err == nil {
			t.Errorf("%s should fail the scrape", name)
		}
	}
}

func TestATargetFormatNothingDerivesYieldsNoPage(t *testing.T) {
	_, scraped, err := newScraper(t, fakeFetch{outcome: succeededWith(t,
		"http://host/", "text/html", "<html><body>text</body></html>",
	)}).Scrape(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
		documentextraction.Format("braille"),
	)
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}
	if scraped {
		t.Fatal("a target format no derivation reaches should yield no page")
	}
}
