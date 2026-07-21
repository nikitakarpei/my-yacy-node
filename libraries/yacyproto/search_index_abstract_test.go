package yacyproto_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func mustParseHash(t *testing.T, raw string) yacymodel.Hash {
	t.Helper()
	hash, err := yacymodel.ParseHash(raw)
	if err != nil {
		t.Fatal(err)
	}

	return hash
}

func TestEncodeSearchIndexAbstractGroupsURLHashes(t *testing.T) {
	t.Parallel()

	got := yacyproto.EncodeSearchIndexAbstract([]yacymodel.Hash{
		mustParseHash(t, "bbbbbbAAAAAA"),
		mustParseHash(t, "aaaaaaBBBBBB"),
		mustParseHash(t, "ccccccAAAAAA"),
	})
	want := "{AAAAAA:bbbbbbcccccc,BBBBBB:aaaaaa}"
	if got != want {
		t.Fatalf("abstract = %q, want %q", got, want)
	}
}

func TestEncodeSearchIndexAbstractEmpty(t *testing.T) {
	t.Parallel()

	if got := yacyproto.EncodeSearchIndexAbstract(nil); got != "{}" {
		t.Fatalf("abstract = %q, want {}", got)
	}
}
