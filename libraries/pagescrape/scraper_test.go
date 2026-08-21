package pagescrape_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
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
	string,
	pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	return f.outcome, f.err
}

func succeededWith(finalURL, contentType, body string) pagefetch.FetchOutcome {
	return pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			FinalURL:    finalURL,
			ContentType: contentType,
			Body:        []byte(body),
		},
	}
}

func newMarkdownScraper(t *testing.T, fetch pagefetch.Fetcher) *pagescrape.Scraper {
	t.Helper()
	scraper, err := pagescrape.New(fetch, contentformatgraph.FormatMarkdown)
	if err != nil {
		t.Fatalf("new scraper: %v", err)
	}
	return scraper
}

func scrape(t *testing.T, outcome pagefetch.FetchOutcome) (pagescrape.ScrapedPage, bool) {
	t.Helper()
	page, scraped, err := newMarkdownScraper(t, fakeFetch{outcome: outcome}).
		Scrape(context.Background(), "http://host/")
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}
	return page, scraped
}

func TestScrapeDerivesTheTargetFormatOfTheFetchedDocument(t *testing.T) {
	page, scraped := scrape(t, succeededWith("http://HOST:80/a", "text/html", originHTML))

	if !scraped {
		t.Fatal("want a scraped page")
	}
	if page.CanonicalURL != "http://host/a" {
		t.Errorf("canonical url = %q, want http://host/a", page.CanonicalURL)
	}
	if !strings.Contains(string(page.Content), "words here") {
		t.Errorf("content = %q, want it to carry the page text", page.Content)
	}
}

func TestScrapeCarriesWhatTheDocumentSaysAboutItself(t *testing.T) {
	page, scraped := scrape(t, succeededWith("http://host/a", "text/html", originHTML))

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
	page, scraped := scrape(t, succeededWith(
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
		_, _, err := newMarkdownScraper(t, fetch).Scrape(context.Background(), "http://host/")
		if err == nil {
			t.Errorf("%s should fail the scrape", name)
		}
	}
}

func TestNewRejectsATargetFormatNothingDerives(t *testing.T) {
	if _, err := pagescrape.New(
		fakeFetch{}, contentformatgraph.Format("braille"),
	); err == nil {
		t.Fatal("a target format no derivation reaches should fail construction")
	}
}
