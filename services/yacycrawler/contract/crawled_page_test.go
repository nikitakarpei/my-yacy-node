package yacycrawlcontract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestCrawledPageRoundTrip(t *testing.T) {
	page := yacycrawlcontract.CrawledPage{
		PageURL: canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
	}

	data, err := yacycrawlcontract.MarshalCrawledPage(page)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := yacycrawlcontract.UnmarshalCrawledPage(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != page {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", page, got)
	}
}

func TestUnmarshalCrawledPageRejectsAPageURLThatIsNotCanonical(t *testing.T) {
	_, err := yacycrawlcontract.UnmarshalCrawledPage(
		[]byte(`{"PageURL":"HTTPS://Example.ORG/a"}`),
	)

	if err == nil {
		t.Fatal("UnmarshalCrawledPage returned nil, want the uncanonical page url failure")
	}
}
