package yacymodel_test

import (
	"reflect"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestPeerHashesOfPreservesSeedOrder(t *testing.T) {
	first := mustParseHash(t, "AAAAAAAAAAAA")
	second := mustParseHash(t, "BBBBBBBBBBBB")
	seeds := []yacymodel.Seed{{Hash: first}, {Hash: second}}

	got := yacymodel.PeerHashesOf(seeds)
	want := []yacymodel.Hash{first, second}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("peer hashes = %v, want %v", got, want)
	}
}
