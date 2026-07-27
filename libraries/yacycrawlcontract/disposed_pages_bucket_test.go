package yacycrawlcontract_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestDisposedPageKeyIsStableHexDigest(t *testing.T) {
	const url = "http://example.com/a"
	got := yacycrawlcontract.DisposedPageKey(url)
	want := "5bd48fa66118084cc32779267a31116dc05c70bcbca0f28e990cd58ce10afeae"
	if got != want {
		t.Fatalf("DisposedPageKey(%q) = %q, want %q", url, got, want)
	}
	if same := yacycrawlcontract.DisposedPageKey(url); same != got {
		t.Fatalf("derivation not deterministic: %q != %q", same, got)
	}
	if other := yacycrawlcontract.DisposedPageKey("http://example.com/b"); other == got {
		t.Fatal("distinct urls derived the same key")
	}
}

func TestEnsureDisposedPagesBucketCreatesStore(t *testing.T) {
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))

	if err := yacycrawlcontract.EnsureDisposedPagesBucket(
		context.Background(),
		js,
		yacycrawlcontract.DisposedPagesBucketSpec{MaxBytes: 1 << 20},
	); err != nil {
		t.Fatalf("ensure disposed pages bucket: %v", err)
	}

	if _, err := js.KeyValue(
		context.Background(),
		yacycrawlcontract.DisposedPagesBucketName,
	); err != nil {
		t.Fatalf("disposed pages bucket: %v", err)
	}
}
