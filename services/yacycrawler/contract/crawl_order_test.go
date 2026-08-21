package yacycrawlcontract_test

import (
	"reflect"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestCrawlOrderRoundTrip(t *testing.T) {
	order := yacycrawlcontract.CrawlOrder{
		OrderID: "3f8a2c14-6b2d-4e1a-9c7f-8d0e1a2b3c4d",
		Profile: yacycrawlcontract.CrawlProfile{
			Name:                   "deep",
			Scope:                  yacycrawlcontract.ScopeSubpath,
			URLMustMatch:           yacycrawlcontract.MatchAll,
			URLMustNotMatch:        ".*\\.pdf",
			MaxDepth:               4,
			AllowQueryURLs:         true,
			MaxPagesPerHost:        100,
			IgnoresIndexingRefusal: true,
		},
		SeedURLs: []canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
			canonicalurltest.CanonicalURLOf(t, "https://example.org/b"),
		},
	}

	data, err := yacycrawlcontract.MarshalCrawlOrder(order)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := yacycrawlcontract.UnmarshalCrawlOrder(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(order, got) {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", order, got)
	}
}
