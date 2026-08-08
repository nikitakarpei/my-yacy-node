package yacycrawlcontract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestRedirectResolutionKeyIsStableHexDigest(t *testing.T) {
	const url = "http://example.com/a"
	got := yacycrawlcontract.RedirectResolutionKey(url)
	want := "5bd48fa66118084cc32779267a31116dc05c70bcbca0f28e990cd58ce10afeae"
	if got != want {
		t.Fatalf("RedirectResolutionKey(%q) = %q, want %q", url, got, want)
	}
	if same := yacycrawlcontract.RedirectResolutionKey(url); same != got {
		t.Fatalf("derivation not deterministic: %q != %q", same, got)
	}
	if other := yacycrawlcontract.RedirectResolutionKey("http://example.com/b"); other == got {
		t.Fatal("distinct urls derived the same key")
	}
}
