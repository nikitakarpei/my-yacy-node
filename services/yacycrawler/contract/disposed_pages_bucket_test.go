package yacycrawlcontract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/canonicalurltest"
)

func TestDisposedPageKeyIsStableHexDigest(t *testing.T) {
	url := canonicalurltest.CanonicalURLOf(t, "http://example.com/a")
	got := yacycrawlcontract.DisposedPageKey(url)
	want := "5bd48fa66118084cc32779267a31116dc05c70bcbca0f28e990cd58ce10afeae"
	if got != want {
		t.Fatalf("DisposedPageKey(%q) = %q, want %q", url.String(), got, want)
	}
	if same := yacycrawlcontract.DisposedPageKey(url); same != got {
		t.Fatalf("derivation not deterministic: %q != %q", same, got)
	}
	if other := yacycrawlcontract.DisposedPageKey(
		canonicalurltest.CanonicalURLOf(t, "http://example.com/b"),
	); other == got {
		t.Fatal("distinct urls derived the same key")
	}
}
