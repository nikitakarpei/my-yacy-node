package yacymodel_test

import (
	"strconv"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestDHTRingPositionOf(t *testing.T) {
	low := yacymodel.DHTRingPositionOf(mustParseHash(t, "AAAAAAAAAAAA"))
	high := yacymodel.DHTRingPositionOf(mustParseHash(t, "__________AA"))
	if low >= high {
		t.Errorf("expected ring order low(%d) < high(%d)", low, high)
	}
	if high != yacymodel.MaxDHTRingPosition {
		t.Errorf(
			"DHTRingPositionOf all-last folded symbols = %d, want %d",
			high, yacymodel.MaxDHTRingPosition,
		)
	}
}

func TestDHTRingPositionOfFoldsSymbolsPastTheTenth(t *testing.T) {
	if yacymodel.DHTRingPositionOf(yacymodel.Hash{}) != yacymodel.DHTRingPositionOf(
		mustParseHash(t, "AAAAAAAAAAAA"),
	) {
		t.Error("the zero hash and all-first symbols must take the same ring position")
	}

	eleventhB := yacymodel.DHTRingPositionOf(mustParseHash(t, "AAAAAAAAAAAB"))
	eleventhZ := yacymodel.DHTRingPositionOf(mustParseHash(t, "AAAAAAAAAAAZ"))
	if eleventhB != eleventhZ {
		t.Errorf(
			"symbols past the tenth must not move the position: %d != %d",
			eleventhB,
			eleventhZ,
		)
	}
}

func TestHashFromDHTRingPositionRoundTrip(t *testing.T) {
	word := mustParseHash(t, "hHJBztzcFn__")
	position := yacymodel.DHTRingPositionOf(word)
	if got := yacymodel.HashFromDHTRingPosition(position); got != word {
		t.Errorf("HashFromDHTRingPosition(DHTRingPositionOf(%v)) = %v, want %v", word, got, word)
	}
}

func TestDHTRingPositionOfPosting(t *testing.T) {
	word := mustParseHash(t, "hHJBztzcFn76")
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatal(err)
	}

	first := postingOfWordForURL(t, word, "AAAAAAAAAAAA")
	second := postingOfWordForURL(t, word, "____________")

	firstPosition := yacymodel.DHTRingPositionOfPosting(first, partitions)
	secondPosition := yacymodel.DHTRingPositionOfPosting(second, partitions)
	if firstPosition == secondPosition {
		t.Errorf(
			"postings of one word for different urls must take different positions: %d == %d",
			firstPosition,
			secondPosition,
		)
	}

	for _, position := range []yacymodel.DHTRingPosition{firstPosition, secondPosition} {
		if distance := position.DistanceFromPostingsOfWord(word, partitions); distance != 0 {
			t.Errorf(
				"a posting position must lie on the postings of its word: distance %d, want 0",
				distance,
			)
		}
	}
}

func TestDHTRingPartitionsFromExponent(t *testing.T) {
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatal(err)
	}
	if partitions != 16 {
		t.Errorf("DHTRingPartitionsFromExponent(4) = %d, want 16", partitions)
	}
	if _, err := yacymodel.DHTRingPartitionsFromExponent(63); err == nil {
		t.Errorf("DHTRingPartitionsFromExponent(63) should fail")
	}
}

func TestDistanceFromPostingsOfWord(t *testing.T) {
	word := mustParseHash(t, "hHJBztzcFn76")
	wholeRing, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatal(err)
	}

	wordPosition := yacymodel.DHTRingPositionOf(word)
	if distance := wordPosition.DistanceFromPostingsOfWord(word, wholeRing); distance != 0 {
		t.Errorf("distance at the word's own position = %d, want 0", distance)
	}

	halfway := wordPosition + yacymodel.MaxDHTRingPosition/2
	fraction := halfway.DistanceFromPostingsOfWord(word, wholeRing).FractionOfDHTRing()
	if fraction < 0.49 || fraction > 0.51 {
		t.Errorf("fraction half a ring past the word = %v, want about 0.5", fraction)
	}
}

func TestDistanceTo(t *testing.T) {
	if distance := yacymodel.DHTRingPosition(10).DistanceTo(40); distance != 30 {
		t.Errorf("DistanceTo forward = %d, want 30", distance)
	}

	wrapped := yacymodel.DHTRingPosition(40).DistanceTo(10)
	want := yacymodel.DHTRingDistance((yacymodel.MaxDHTRingPosition-40)+10) + 1
	if wrapped != want {
		t.Errorf("DistanceTo wrapping the ring = %d, want %d", wrapped, want)
	}

	if distance := yacymodel.DHTRingPosition(5).DistanceTo(5); distance != 0 {
		t.Errorf("DistanceTo the same position = %d, want 0", distance)
	}
}

func TestFractionOfDHTRing(t *testing.T) {
	if fraction := yacymodel.DHTRingDistance(0).FractionOfDHTRing(); fraction != 0 {
		t.Errorf("FractionOfDHTRing of no distance = %v, want 0", fraction)
	}

	half := yacymodel.DHTRingDistance(yacymodel.MaxDHTRingPosition / 2).FractionOfDHTRing()
	if half < 0.49 || half > 0.51 {
		t.Errorf("FractionOfDHTRing of half the ring = %v, want about 0.5", half)
	}
}

func postingOfWordForURL(
	t *testing.T,
	word yacymodel.Hash,
	rawURLHash string,
) yacymodel.RWIPosting {
	t.Helper()

	urlHash, err := yacymodel.ParseURLHash(rawURLHash)
	if err != nil {
		t.Fatal(err)
	}

	return yacymodel.RWIPosting{WordHash: word, URLHash: urlHash}
}

func TestDHTRingPositionOfWordInPartitionHoldsEveryPostingOfThatWord(t *testing.T) {
	const partitionExponent = 4
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(partitionExponent)
	if err != nil {
		t.Fatalf("dht ring partitions: %v", err)
	}
	word := yacymodel.WordHash("partitionprobe")

	named := map[yacymodel.DHTRingPosition]struct{}{}
	for partition := range uint(partitions) {
		named[yacymodel.DHTRingPositionOfWordInPartition(word, partition, partitions)] = struct{}{}
	}
	if len(named) != int(partitions) {
		t.Fatalf("partitions name %d distinct positions, want %d", len(named), partitions)
	}

	for document := range 4096 {
		urlHash, err := yacymodel.ParseURLHash(
			yacymodel.WordHash(strconv.Itoa(document)).String(),
		)
		if err != nil {
			t.Fatalf("url hash: %v", err)
		}
		position := yacymodel.DHTRingPositionOfPosting(
			yacymodel.RWIPosting{WordHash: word, URLHash: urlHash},
			partitions,
		)
		if _, ok := named[position]; !ok {
			t.Fatalf("posting of the word sits at %d, which no partition names", position)
		}
	}
}
