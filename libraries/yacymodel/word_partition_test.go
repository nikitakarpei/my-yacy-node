package yacymodel_test

import (
	"strconv"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestEveryPostingOfAWordPartitionSitsAtItsPosition(t *testing.T) {
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatalf("dht ring partitions: %v", err)
	}
	word := yacymodel.WordHash("wordpartition")

	for document := range 256 {
		url := urlHashOf(t, document)
		posting := yacymodel.RWIPosting{WordHash: word, URLHash: url}
		wordPartition := yacymodel.WordPartitionOfWordAndURL(word, url, partitions)

		if wordPartition.Word != word || wordPartition.Partition != partitions.PartitionOf(url) {
			t.Fatalf("word partition of the posting on %v = %+v", url, wordPartition)
		}
		if got, want := yacymodel.DHTRingPositionOfWordPartition(
			wordPartition,
			partitions,
		), yacymodel.DHTRingPositionOfPosting(posting, partitions); got != want {
			t.Fatalf(
				"word partition of the posting on %v sits at %d, the posting at %d",
				url,
				got,
				want,
			)
		}
	}
}

func TestWordPartitionsOfOneWordDifferPerPartition(t *testing.T) {
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatalf("dht ring partitions: %v", err)
	}
	word := yacymodel.WordHash("wordpartition")

	positions := map[yacymodel.DHTRingPosition]struct{}{}
	for partition := range partitions.Amount() {
		positions[yacymodel.DHTRingPositionOfWordPartition(
			yacymodel.WordPartition{Word: word, Partition: partition},
			partitions,
		)] = struct{}{}
	}
	if len(positions) != int(partitions.Amount()) {
		t.Fatalf(
			"word partitions sit at %d distinct positions, want %d",
			len(positions),
			partitions.Amount(),
		)
	}
}

func urlHashOf(t *testing.T, document int) yacymodel.URLHash {
	t.Helper()

	url, err := yacymodel.ParseURLHash(yacymodel.WordHash(strconv.Itoa(document)).String())
	if err != nil {
		t.Fatalf("url hash: %v", err)
	}

	return url
}
