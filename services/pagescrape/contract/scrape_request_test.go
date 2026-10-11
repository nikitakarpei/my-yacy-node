package pagescrapecontract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

func TestScrapeRequestRoundTrip(t *testing.T) {
	request := pagescrapecontract.ScrapeRequest{
		PageURL:  canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
		FetchURL: canonicalurltest.CanonicalURLOf(t, "https://archive.example/a"),
	}

	data, err := pagescrapecontract.MarshalScrapeRequest(request)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := pagescrapecontract.UnmarshalScrapeRequest(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != request {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", request, got)
	}
}

func TestScrapeRequestWithoutAFetchURLIsReadFromThePageURL(t *testing.T) {
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/a")

	data, err := pagescrapecontract.MarshalScrapeRequest(
		pagescrapecontract.ScrapeRequest{PageURL: pageURL},
	)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := pagescrapecontract.UnmarshalScrapeRequest(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.FetchURL != pageURL {
		t.Errorf("fetch url = %q, want the page url %q", got.FetchURL, pageURL)
	}
}

func TestUnmarshalScrapeRequestInvalidJSON(t *testing.T) {
	if _, err := pagescrapecontract.UnmarshalScrapeRequest([]byte("not json")); err == nil {
		t.Fatal("want an error for invalid JSON")
	}
}

func TestScrapeRequestCarriesTheDeferralTerms(t *testing.T) {
	request := pagescrapecontract.ScrapeRequest{
		PageURL:           canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
		GivesUpOnDeferral: true,
	}

	data, err := pagescrapecontract.MarshalScrapeRequest(request)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := pagescrapecontract.UnmarshalScrapeRequest(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got.GivesUpOnDeferral {
		t.Error("gives up on deferral = false, want true")
	}
}
