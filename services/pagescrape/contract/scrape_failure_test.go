package pagescrapecontract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

func TestScrapeFailureRoundTrip(t *testing.T) {
	failure := pagescrapecontract.ScrapeFailure{
		PageURL:  canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
		FetchURL: canonicalurltest.CanonicalURLOf(t, "https://archive.example/a"),
		Reason:   pagescrapecontract.NoReasonGiven,
	}

	data, err := pagescrapecontract.MarshalScrapeFailure(failure)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := pagescrapecontract.UnmarshalScrapeFailure(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != failure {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", failure, got)
	}
}

func TestUnmarshalScrapeFailureWithoutAReason(t *testing.T) {
	data, err := pagescrapecontract.MarshalScrapeFailure(pagescrapecontract.ScrapeFailure{
		PageURL: canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := pagescrapecontract.UnmarshalScrapeFailure(data); err == nil {
		t.Fatal("want an error for a failure that names no reason")
	}
}

func TestUnmarshalScrapeFailureInvalidJSON(t *testing.T) {
	if _, err := pagescrapecontract.UnmarshalScrapeFailure([]byte("not json")); err == nil {
		t.Fatal("want an error for invalid JSON")
	}
}
