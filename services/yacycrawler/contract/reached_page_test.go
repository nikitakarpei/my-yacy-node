package yacycrawlcontract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestReachedPageRoundTrip(t *testing.T) {
	page := yacycrawlcontract.ReachedPage{
		CanonicalURL: canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
	}

	data, err := yacycrawlcontract.MarshalReachedPage(page)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := yacycrawlcontract.UnmarshalReachedPage(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != page {
		t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", page, got)
	}
}

func TestUnmarshalReachedPageInvalidJSON(t *testing.T) {
	if _, err := yacycrawlcontract.UnmarshalReachedPage([]byte("not json")); err == nil {
		t.Fatal("want an error for invalid JSON")
	}
}
