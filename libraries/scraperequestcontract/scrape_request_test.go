package scraperequestcontract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
)

func TestScrapeRequestRoundTrip(t *testing.T) {
	request := scraperequestcontract.ScrapeRequest{
		CanonicalURL: canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
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

func TestUnmarshalScrapeRequestInvalidJSON(t *testing.T) {
	if _, err := scraperequestcontract.UnmarshalScrapeRequest([]byte("not json")); err == nil {
		t.Fatal("want an error for invalid JSON")
	}
}
