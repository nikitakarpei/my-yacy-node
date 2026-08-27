package scraperequestcontract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
)

func TestScrapeRequestRoundTrip(t *testing.T) {
	request := scraperequestcontract.ScrapeRequest{
		PageURL:  canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
		FetchURL: canonicalurltest.CanonicalURLOf(t, "https://archive.example/a"),
	}

	data, err := scraperequestcontract.MarshalScrapeRequest(request)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := scraperequestcontract.UnmarshalScrapeRequest(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != request {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", request, got)
	}
}

func TestScrapeRequestWithoutAFetchURLIsReadFromThePageURL(t *testing.T) {
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/a")

	data, err := scraperequestcontract.MarshalScrapeRequest(
		scraperequestcontract.ScrapeRequest{PageURL: pageURL},
	)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := scraperequestcontract.UnmarshalScrapeRequest(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.FetchURL != pageURL {
		t.Errorf("fetch url = %q, want the page url %q", got.FetchURL, pageURL)
	}
}

func TestUnmarshalScrapeRequestInvalidJSON(t *testing.T) {
	if _, err := scraperequestcontract.UnmarshalScrapeRequest([]byte("not json")); err == nil {
		t.Fatal("want an error for invalid JSON")
	}
}
