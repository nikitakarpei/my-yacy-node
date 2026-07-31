package yacyproto_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func mustParseURLHash(t *testing.T, raw string) yacymodel.URLHash {
	t.Helper()
	hash, err := yacymodel.ParseURLHash(raw)
	if err != nil {
		t.Fatal(err)
	}

	return hash
}

func TestEncodeSearchIndexAbstractGroupsURLHashes(t *testing.T) {
	t.Parallel()

	got := yacyproto.EncodeSearchIndexAbstract([]yacymodel.URLHash{
		mustParseURLHash(t, "bbbbbbAAAAAA"),
		mustParseURLHash(t, "aaaaaaBBBBBB"),
		mustParseURLHash(t, "ccccccAAAAAA"),
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
