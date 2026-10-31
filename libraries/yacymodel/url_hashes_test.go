package yacymodel_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestURLHashesContainEachHashOnce(t *testing.T) {
	first := hashOfAddress(t, "https://first.example/")
	second := hashOfAddress(t, "https://second.example/")

	hashes := yacymodel.URLHashesFrom(first, second, first)

	if hashes.Amount() != 2 || !hashes.Contains(first) || !hashes.Contains(second) {
		t.Fatalf("the hashes hold %v, want the two hashes once each", hashes.InHashOrder())
	}
}

func TestURLHashesDoNotContainAHashNeverAdded(t *testing.T) {
	hashes := yacymodel.URLHashesFrom(hashOfAddress(t, "https://first.example/"))

	if hashes.Contains(hashOfAddress(t, "https://second.example/")) {
		t.Fatal("the hashes contain a hash never added")
	}
}

func TestURLHashesComeInHashOrder(t *testing.T) {
	hashes := yacymodel.URLHashesFrom(
		urlHashFrom(
			t,
			"zzzzzzAAAAAA",
		),
		urlHashFrom(t, "AAAAAAzzzzzz"),
		urlHashFrom(t, "mmmmmmmmmmmm"),
	)

	got := slices.Collect(hashes.All())

	if !slices.IsSortedFunc(got, yacymodel.URLHash.CompareInAlphabetOrder) {
		t.Fatalf("the hashes come as %v, want them in hash order", got)
	}
	if !slices.Equal(got, hashes.InHashOrder()) {
		t.Fatalf("all hashes %v differ from the hashes in hash order %v", got, hashes.InHashOrder())
	}
}

func TestTheUnionOfURLHashesHoldsEachHashOfAllOnce(t *testing.T) {
	first := urlHashFrom(t, "AAAAAAAAAAAA")
	second := urlHashFrom(t, "BBBBBBBBBBBB")
	third := urlHashFrom(t, "CCCCCCCCCCCC")

	union := yacymodel.UnionOf(
		yacymodel.URLHashesFrom(first, second),
		yacymodel.URLHashesFrom(second, third),
		yacymodel.URLHashesFrom(third),
	)

	if want := []yacymodel.URLHash{first, second, third}; !slices.Equal(union.InHashOrder(), want) {
		t.Fatalf("the union holds %v, want %v", union.InHashOrder(), want)
	}
}

func TestURLHashesWithoutOthersKeepOnlyTheirOwnHashes(t *testing.T) {
	first := urlHashFrom(t, "AAAAAAAAAAAA")
	second := urlHashFrom(t, "BBBBBBBBBBBB")

	kept := yacymodel.URLHashesFrom(first, second).Without(yacymodel.URLHashesFrom(second))

	if want := []yacymodel.URLHash{first}; !slices.Equal(kept.InHashOrder(), want) {
		t.Fatalf("the hashes without the others are %v, want %v", kept.InHashOrder(), want)
	}
}

func TestURLHashesBuilderStartsOverAfterItHandsOverTheHashes(t *testing.T) {
	var builder yacymodel.URLHashesBuilder
	builder.Add(urlHashFrom(t, "AAAAAAAAAAAA"))
	first := builder.URLHashes()
	builder.Add(urlHashFrom(t, "BBBBBBBBBBBB"))

	second := builder.URLHashes()

	if first.Amount() != 1 || second.Amount() != 1 ||
		first.Contains(urlHashFrom(t, "BBBBBBBBBBBB")) {
		t.Fatalf("the builder handed over %v and %v, want one hash each",
			first.InHashOrder(), second.InHashOrder())
	}
}

func TestURLHashesInAPartitionAreTheHashesThePartitionHolds(t *testing.T) {
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatalf("dht ring partitions: %v", err)
	}
	hashes := yacymodel.URLHashesFrom(
		partitions.LowestURLHashIn(2), partitions.HighestURLHashIn(2),
		partitions.HighestURLHashIn(1), partitions.LowestURLHashIn(3),
		urlHashFrom(t, "AAAAAAAAAAAA"), urlHashFrom(t, "____________"),
	)

	for partition := range partitions.Amount() {
		for hash := range partitions.URLHashesIn(partition, hashes).All() {
			if got := partitions.PartitionOf(hash); got != partition {
				t.Fatalf("hash %v of partition %d sits in %d", hash, partition, got)
			}
		}
	}
	if got := partitions.URLHashesIn(2, hashes).Amount(); got != 2 {
		t.Fatalf("partition 2 holds %d hashes, want its lowest and highest", got)
	}
}

func urlHashFrom(t *testing.T, raw string) yacymodel.URLHash {
	t.Helper()
	hash, err := yacymodel.ParseURLHash(raw)
	if err != nil {
		t.Fatalf("parse url hash %q: %v", raw, err)
	}

	return hash
}
