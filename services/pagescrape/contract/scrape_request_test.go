package pagescrapecontract_test

import (
	"strings"
	"testing"
	"time"

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

func TestScrapeScheduleSubjectOfIsUniquePerPage(t *testing.T) {
	subject := pagescrapecontract.ScrapeScheduleSubjectOf(
		canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
	)
	other := pagescrapecontract.ScrapeScheduleSubjectOf(
		canonicalurltest.CanonicalURLOf(t, "https://example.org/b"),
	)

	if !strings.HasPrefix(subject, pagescrapecontract.ScrapeScheduleSubjectPrefix+".") {
		t.Errorf("subject %q does not start with the schedule prefix", subject)
	}
	if subject == other {
		t.Errorf("two pages share the subject %q", subject)
	}
}

func TestScrapeRequestCarriesTheDeferralTerms(t *testing.T) {
	request := pagescrapecontract.ScrapeRequest{
		PageURL:           canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
		DeferredSince:     time.Date(2026, time.September, 2, 10, 0, 0, 0, time.UTC),
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
	if !got.DeferredSince.Equal(request.DeferredSince) {
		t.Errorf("deferred since = %v, want %v", got.DeferredSince, request.DeferredSince)
	}
	if !got.GivesUpOnDeferral {
		t.Error("gives up on deferral = false, want true")
	}
}
