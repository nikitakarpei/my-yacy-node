package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestURLHashesContainEachAddedHashOnce(t *testing.T) {
	first := hashOfAddress(t, "https://first.example/")
	second := hashOfAddress(t, "https://second.example/")
	hashes := yacymodel.URLHashes{}

	hashes.AddEach([]yacymodel.URLHash{first, second, first})

	if len(hashes) != 2 || !hashes.Contains(first) || !hashes.Contains(second) {
		t.Fatalf("the hashes hold %v, want the two added hashes once each", hashes)
	}
}

func TestURLHashesDoNotContainAHashNeverAdded(t *testing.T) {
	hashes := yacymodel.URLHashes{}
	hashes.Add(hashOfAddress(t, "https://first.example/"))

	if hashes.Contains(hashOfAddress(t, "https://second.example/")) {
		t.Fatal("the hashes contain a hash never added")
	}
}

func TestURLHashesInHashOrderComeSortedByTheirHashes(t *testing.T) {
	first := hashOfAddress(t, "https://first.example/")
	second := hashOfAddress(t, "https://second.example/")
	hashes := yacymodel.URLHashes{first: {}, second: {}}

	got := hashes.InHashOrder()

	if len(got) != 2 || got[0].String() > got[1].String() {
		t.Fatalf("the hashes in hash order are %v, want them sorted by hash", got)
	}
}
