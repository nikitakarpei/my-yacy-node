package documentamounts_test

import (
	"fmt"
	"maps"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentamounts"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type answeredWordPartition struct {
	spelledWord string
	partition   uint
	answers     []wordpartitionasks.ReplicaAnswer
}

func answeredFor(
	spelledWord string,
	partition uint,
	answers ...wordpartitionasks.ReplicaAnswer,
) answeredWordPartition {
	return answeredWordPartition{spelledWord: spelledWord, partition: partition, answers: answers}
}

func answeredForBothWords(
	partition uint,
	answers ...wordpartitionasks.ReplicaAnswer,
) []answeredWordPartition {
	return []answeredWordPartition{
		answeredFor(firstWord, partition, answers...),
		answeredFor(secondWord, partition, answers...),
	}
}

func measurementOf(
	partitions yacymodel.DHTRingPartitions,
	answeredWordPartitions ...[]answeredWordPartition,
) *documentamounts.Measurement {
	measurement := documentamounts.NoneMeasuredYet(query, partitions)
	for _, answered := range answeredWordPartitions {
		for _, wordPartition := range answered {
			measurement.WordPartitionAnswered(
				yacymodel.WordHash(wordPartition.spelledWord),
				wordPartition.partition,
				wordPartition.answers,
			)
		}
	}

	return measurement
}

func counted(amountHeld int) wordpartitionasks.ReplicaAnswer {
	return answerCounting(amountHeld)
}

func distinctDocuments(t *testing.T, amount int) []yacymodel.URLHash {
	t.Helper()

	documents := make([]yacymodel.URLHash, 0, amount)
	for place := range amount {
		document, err := yacymodel.URLHashOf(fmt.Sprintf("https://document-%d.example/", place))
		if err != nil {
			t.Fatalf("hash a document address: %v", err)
		}
		documents = append(documents, document)
	}

	return documents
}

func answerCounting(
	amountHeld int,
	documents ...yacymodel.URLHash,
) wordpartitionasks.ReplicaAnswer {
	answer := wordpartitionasks.ReplicaAnswer{AmountOfDocumentsHeld: yacymodel.Some(amountHeld)}
	for _, document := range documents {
		answer.ListedDocuments = append(
			answer.ListedDocuments, wordpartitionasks.ListedDocument{Hash: document},
		)
	}

	return answer
}

func heldForBothWords(amountHeld int) map[yacymodel.Hash]int {
	return amountOfEachWord(map[string]int{firstWord: amountHeld, secondWord: amountHeld})
}

func TestEveryPartitionOfAQueryWordAddsWhatItsReplicasCounted(t *testing.T) {
	t.Parallel()

	measurement := measurementOf(
		twoPartitionsOfTheRing,
		answeredForBothWords(0, counted(100), counted(100), counted(100)),
		answeredForBothWords(1, counted(10)),
	)

	if got, want := measurement.HeldPerQueryWord(), heldForBothWords(110); !maps.Equal(got, want) {
		t.Fatalf("the measurement estimates %v held, want %v", got, want)
	}
}

func TestAPartitionOfAnEvenAmountOfCountsTakesTheLowerMiddleOne(t *testing.T) {
	t.Parallel()

	measurement := measurementOf(1, answeredForBothWords(0, counted(10), counted(20)))

	if got, want := measurement.HeldPerQueryWord(), heldForBothWords(10); !maps.Equal(got, want) {
		t.Fatalf("the measurement estimates %v held, want %v", got, want)
	}
}

func TestAPartitionNoPeerCountedTakesTheMiddleOfThePartitionsThatWereCounted(t *testing.T) {
	t.Parallel()

	measurement := measurementOf(
		3,
		answeredForBothWords(0, counted(10)),
		answeredForBothWords(1, counted(30)),
		answeredForBothWords(2, wordpartitionasks.ReplicaAnswer{}),
	)

	if got, want := measurement.HeldPerQueryWord(), heldForBothWords(10+30+10); !maps.Equal(
		got, want,
	) {
		t.Fatalf("the measurement estimates %v held, want %v", got, want)
	}
}

func TestAQueryWordNoPeerCountedCarriesNoDocumentsHeld(t *testing.T) {
	t.Parallel()

	measurement := measurementOf(
		twoPartitionsOfTheRing,
		answeredForBothWords(0, wordpartitionasks.ReplicaAnswer{}),
		answeredForBothWords(1, wordpartitionasks.ReplicaAnswer{}),
	)

	if got := measurement.HeldPerQueryWord(); len(got) != 0 {
		t.Fatalf("the measurement estimates %v held, want none", got)
	}
}

func TestTheAmountInAPartitionIsTheMiddleOfThePartitionsThatWereCounted(t *testing.T) {
	t.Parallel()

	measurement := measurementOf(
		4,
		answeredForBothWords(0, counted(10)),
		answeredForBothWords(1, counted(30), counted(50)),
		answeredForBothWords(2, counted(20)),
		answeredForBothWords(3, wordpartitionasks.ReplicaAnswer{}),
	)

	if got, want := measurement.InAPartitionPerQueryWord(), heldForBothWords(20); !maps.Equal(
		got, want,
	) {
		t.Fatalf("the measurement gives %v in a partition, want %v", got, want)
	}
}

func TestAQueryWordNoPeerCountedHasNoAmountInAPartition(t *testing.T) {
	t.Parallel()

	measurement := measurementOf(twoPartitionsOfTheRing, []answeredWordPartition{
		answeredFor(firstWord, 0, wordpartitionasks.ReplicaAnswer{}),
		answeredFor(secondWord, 0, counted(10)),
	})

	want := amountOfEachWord(map[string]int{secondWord: 10})
	if got := measurement.InAPartitionPerQueryWord(); !maps.Equal(got, want) {
		t.Fatalf("the measurement gives %v in a partition, want %v", got, want)
	}
}

func TestAnAnsweredCompoundWordIsNoQueryWord(t *testing.T) {
	t.Parallel()

	documents := distinctDocuments(t, 2)
	measurement := measurementOf(1, []answeredWordPartition{
		answeredFor(firstWord, 0, answerCounting(1, documents[0])),
		answeredFor(firstWord+secondWord, 0, answerCounting(1, documents[1])),
	})

	if measurement.AmountOfQueryWordsHeldByNoPeer() != 1 {
		t.Fatalf(
			"the measurement counts %d query words held by no peer, want only the second word",
			measurement.AmountOfQueryWordsHeldByNoPeer(),
		)
	}
}

func TestTheAmountListedOfAQueryWordCountsEachDocumentItsReplicasListedOnce(t *testing.T) {
	t.Parallel()

	documents := distinctDocuments(t, 3)
	measurement := measurementOf(2, []answeredWordPartition{
		answeredFor(firstWord, 0, answerCounting(3, documents...)),
		answeredFor(firstWord, 1, answerCounting(1, documents[0])),
	})

	if measurement.AmountListedOf(yacymodel.WordHash(firstWord)) != 3 {
		t.Fatalf(
			"the first word lists %d documents, want the 3 its replicas listed",
			measurement.AmountListedOf(yacymodel.WordHash(firstWord)),
		)
	}
}
