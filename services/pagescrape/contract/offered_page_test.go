package pagescrapecontract_test

import (
	"reflect"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

func TestOfferedPageFromCarriesTheBytesTheFetchReturned(t *testing.T) {
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/a")

	offered := pagescrapecontract.OfferedPageFrom(
		pagescrapecontract.ScrapeRequest{PageURL: pageURL, FetchURL: pageURL},
		pagefetch.FetchedPage{
			ContentType: "text/html",
			Body:        []byte("hello"),
		},
		pageURL,
	)

	if offered.ContentType != "text/html" || string(offered.Body) != "hello" {
		t.Errorf("offered page = %+v", offered)
	}
}

func TestOfferedPageFromKeepsThePageURLTheRequestNamed(t *testing.T) {
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/a")
	landedURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/b")

	offered := pagescrapecontract.OfferedPageFrom(
		pagescrapecontract.ScrapeRequest{PageURL: pageURL, FetchURL: pageURL},
		pagefetch.FetchedPage{},
		landedURL,
	)

	if offered.PageURL != pageURL {
		t.Errorf("page url = %q, want the named page url %q", offered.PageURL, pageURL)
	}
	if offered.LandedURL != landedURL {
		t.Errorf("landed url = %q, want %q", offered.LandedURL, landedURL)
	}
}

func TestOfferedPageFromKeepsThePageURLWhenTheFetchURLNamesAnotherAddress(t *testing.T) {
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/a")
	landedURL := canonicalurltest.CanonicalURLOf(t, "https://archive.example/b")

	offered := pagescrapecontract.OfferedPageFrom(
		pagescrapecontract.ScrapeRequest{
			PageURL:  pageURL,
			FetchURL: canonicalurltest.CanonicalURLOf(t, "https://archive.example/a"),
		},
		pagefetch.FetchedPage{},
		landedURL,
	)

	if offered.PageURL != pageURL {
		t.Errorf("page url = %q, want the named page url %q", offered.PageURL, pageURL)
	}
	if offered.LandedURL != landedURL {
		t.Errorf("landed url = %q, want %q", offered.LandedURL, landedURL)
	}
}

func TestOfferedPageRoundTrip(t *testing.T) {
	page := pagescrapecontract.OfferedPage{
		PageURL:          canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
		LandedURL:        canonicalurltest.CanonicalURLOf(t, "https://example.org/b"),
		ContentType:      "text/html",
		Body:             []byte("hello"),
		RobotsDirectives: []string{"noindex"},
	}

	data, err := pagescrapecontract.MarshalOfferedPage(page)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := pagescrapecontract.UnmarshalOfferedPage(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, page) {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", page, got)
	}
}

func TestUnmarshalOfferedPageInvalidJSON(t *testing.T) {
	if _, err := pagescrapecontract.UnmarshalOfferedPage([]byte("not json")); err == nil {
		t.Fatal("want an error for invalid JSON")
	}
}
