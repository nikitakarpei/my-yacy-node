package yacymodel_test

import (
	"math/big"
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
	for _, exponent := range []uint{0, 4, 62} {
		partitions, err := yacymodel.DHTRingPartitionsFromExponent(exponent)
		if err != nil {
			t.Fatalf("DHTRingPartitionsFromExponent(%d): %v", exponent, err)
		}
		if got, want := partitions.Amount(), uint(1)<<exponent; got != want {
			t.Errorf(
				"DHTRingPartitionsFromExponent(%d).Amount() = %d, want %d",
				exponent,
				got,
				want,
			)
		}
	}
	if _, err := yacymodel.DHTRingPartitionsFromExponent(63); err == nil {
		t.Errorf("DHTRingPartitionsFromExponent(63) should fail")
	}
}

func TestZeroDHTRingPartitionsIsTheWholeRing(t *testing.T) {
	wholeRing, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatal(err)
	}
	var zero yacymodel.DHTRingPartitions
	if zero != wholeRing {
		t.Errorf("zero DHTRingPartitions = %v, want the partitions of exponent 0", zero)
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

func TestPeersHoldingOneWordOfCountsEveryPartitionOfTheRing(t *testing.T) {
	const (
		partitionExponent = 4
		networkRedundancy = 3
	)
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(partitionExponent)
	if err != nil {
		t.Fatalf("dht ring partitions: %v", err)
	}

	holders := yacymodel.PeersHoldingOneWordOf(partitions, networkRedundancy)

	if holders != int(partitions.Amount())*networkRedundancy {
		t.Fatalf(
			"PeersHoldingOneWordOf = %d, want %d peers for each of the %d partitions",
			holders,
			networkRedundancy,
			partitions.Amount(),
		)
	}
}

func TestDHTRingPositionOfWordInPartitionHoldsEveryPostingOfThatWord(t *testing.T) {
	const partitionExponent = 4
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(partitionExponent)
	if err != nil {
		t.Fatalf("dht ring partitions: %v", err)
	}
	word := yacymodel.WordHash("partitionprobe")

	named := map[yacymodel.DHTRingPosition]struct{}{}
	for partition := range partitions.Amount() {
		named[yacymodel.DHTRingPositionOfWordInPartition(word, partition, partitions)] = struct{}{}
	}
	if len(named) != int(partitions.Amount()) {
		t.Fatalf("partitions name %d distinct positions, want %d", len(named), partitions.Amount())
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

func TestPartitionOfPutsThePostingOfEveryWordOnTheURLInThatPartition(t *testing.T) {
	for exponent := range uint(5) {
		partitions, err := yacymodel.DHTRingPartitionsFromExponent(exponent)
		if err != nil {
			t.Fatalf("dht ring partitions: %v", err)
		}
		for wordIndex := range 8 {
			word := yacymodel.WordHash("word" + strconv.Itoa(wordIndex))
			for document := range 256 {
				url, err := yacymodel.ParseURLHash(
					yacymodel.WordHash(strconv.Itoa(document)).String(),
				)
				if err != nil {
					t.Fatalf("url hash: %v", err)
				}
				partition := partitions.PartitionOf(url)

				posting := yacymodel.DHTRingPositionOfWordAndURL(word, url, partitions)
				wordInPartition := yacymodel.DHTRingPositionOfWordInPartition(
					word,
					partition,
					partitions,
				)
				if posting != wordInPartition {
					t.Fatalf(
						"%d partitions: posting of %v on %v sits at %d, partition %d names %d",
						partitions.Amount(), word, url, posting, partition, wordInPartition,
					)
				}
			}
		}
	}
}

func TestPartitionOfTheWholeRingIsTheFirstPartition(t *testing.T) {
	wholeRing, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatalf("dht ring partitions: %v", err)
	}
	url, err := yacymodel.ParseURLHash("____________")
	if err != nil {
		t.Fatalf("url hash: %v", err)
	}

	if partition := wholeRing.PartitionOf(url); partition != 0 {
		t.Fatalf("PartitionOf on the whole ring = %d, want 0", partition)
	}
}

func TestLowestAndHighestURLHashInAPartitionMapBackToThatPartition(t *testing.T) {
	for _, exponent := range []uint{0, 4, 7} {
		partitions, err := yacymodel.DHTRingPartitionsFromExponent(exponent)
		if err != nil {
			t.Fatalf("dht ring partitions: %v", err)
		}
		last := partitions.Amount() - 1
		for _, partition := range []uint{0, last / 2, last} {
			lowest := partitions.LowestURLHashIn(partition)
			highest := partitions.HighestURLHashIn(partition)

			if got := partitions.PartitionOf(lowest); got != partition {
				t.Fatalf("%d partitions: lowest %v sits in %d, want %d",
					partitions.Amount(), lowest, got, partition)
			}
			if got := partitions.PartitionOf(highest); got != partition {
				t.Fatalf("%d partitions: highest %v sits in %d, want %d",
					partitions.Amount(), highest, got, partition)
			}
		}
	}
}

func TestURLHashesOfThePartitionsCoverTheWholeHashRange(t *testing.T) {
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatalf("dht ring partitions: %v", err)
	}
	last := partitions.Amount() - 1

	if lowest := partitions.LowestURLHashIn(0).String(); lowest != "AAAAAAAAAAAA" {
		t.Fatalf("lowest url hash of the first partition = %s", lowest)
	}
	if highest := partitions.HighestURLHashIn(last).String(); highest != "____________" {
		t.Fatalf("highest url hash of the last partition = %s", highest)
	}
	for partition := range last {
		highest := new(big.Int).SetBytes(partitions.HighestURLHashIn(partition).Bytes())
		nextLowest := new(big.Int).SetBytes(partitions.LowestURLHashIn(partition + 1).Bytes())

		if successor := highest.Add(highest, big.NewInt(1)); successor.Cmp(nextLowest) != 0 {
			t.Fatalf("highest url hash of %d is not just before the lowest of %d",
				partition, partition+1)
		}
	}
}
