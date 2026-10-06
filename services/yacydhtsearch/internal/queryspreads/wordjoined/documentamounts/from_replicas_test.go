package documentamounts_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentamounts"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	firstWord              = "berlin"
	secondWord             = "weather"
	twoPartitionsOfTheRing = 2
	askedPartition         = 1
	otherPartition         = 0
)

var query = queryreading.QueryFrom(firstWord+" "+secondWord, "")

type documentAsksOfTheNetwork struct {
	answerOfEachWord map[string]wordpartitionasks.ReplicaAnswer
	partitionsAsked  []uint
}

func documentAsksAnswering(
	answerOfEachWord map[string]wordpartitionasks.ReplicaAnswer,
) *documentAsksOfTheNetwork {
	return &documentAsksOfTheNetwork{answerOfEachWord: answerOfEachWord}
}

func (documentAsks *documentAsksOfTheNetwork) WhichDocumentsHaveIn(
	partition uint,
	words []yacymodel.Hash,
) []wordpartitionasks.SettledAsk {
	documentAsks.partitionsAsked = append(documentAsks.partitionsAsked, partition)
	var answered []wordpartitionasks.SettledAsk
	for spelledWord, answer := range documentAsks.answerOfEachWord {
		if !slices.Contains(words, yacymodel.WordHash(spelledWord)) {
			continue
		}
		answered = append(answered, wordpartitionasks.SettledAsk{
			Ask: wordpartitionasks.Ask{
				Word:      yacymodel.WordHash(spelledWord),
				Partition: partition,
			},
			Answers: []wordpartitionasks.ReplicaAnswer{answer},
		})
	}

	return answered
}

func (documentAsks *documentAsksOfTheNetwork) amountsCountedFromReplicas(
	t *testing.T,
	observer documentamounts.FromReplicasObserver,
) map[yacymodel.Hash]int {
	t.Helper()

	return amountsFromReplicas(observer).AmountsInAPartitionFor(t.Context(), query, documentAsks)
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

func completeAbstractIn(partition uint, amount int) wordpartitionasks.ReplicaAnswer {
	return answerCounting(amount, documentsIn(partition, amount)...)
}

func documentsIn(partition uint, amount int) []yacymodel.URLHash {
	partitions := yacymodel.DHTRingPartitions(twoPartitionsOfTheRing)
	documents := make([]yacymodel.URLHash, 0, amount)
	for place := 0; len(documents) < amount; place++ {
		document, err := yacymodel.URLHashOf(fmt.Sprintf("https://document-%d.example/", place))
		if err != nil || partitions.PartitionOf(document) != partition {
			continue
		}
		documents = append(documents, document)
	}

	return documents
}

type countsFromReplicas struct {
	performed []documentamounts.PerformedFromReplicas
}

func (counts *countsFromReplicas) AmountsCountedFromReplicas(
	_ context.Context,
	performed documentamounts.PerformedFromReplicas,
) {
	counts.performed = append(counts.performed, performed)
}

func amountsFromReplicas(
	observer documentamounts.FromReplicasObserver,
) documentamounts.FromReplicas {
	return documentamounts.NewFromReplicas(
		twoPartitionsOfTheRing,
		func(amountOfPartitions uint) uint { return amountOfPartitions - 1 },
		observer,
	)
}

func amountOfEachWord(amounts map[string]int) map[yacymodel.Hash]int {
	amountsPerWord := make(map[yacymodel.Hash]int, len(amounts))
	for spelledWord, amount := range amounts {
		amountsPerWord[yacymodel.WordHash(spelledWord)] = amount
	}

	return amountsPerWord
}

func TestEachWordIsCountedInThePartitionAsked(t *testing.T) {
	t.Parallel()

	documentAsks := documentAsksAnswering(map[string]wordpartitionasks.ReplicaAnswer{
		firstWord:  completeAbstractIn(askedPartition, 3),
		secondWord: completeAbstractIn(askedPartition, 1),
	})

	amounts := documentAsks.amountsCountedFromReplicas(t, &countsFromReplicas{})

	want := amountOfEachWord(map[string]int{firstWord: 3, secondWord: 1})
	if !maps.Equal(amounts, want) ||
		!slices.Equal(documentAsks.partitionsAsked, []uint{askedPartition}) {
		t.Fatalf(
			"the replicas counted %v after asking %v, want %v in partition %d",
			amounts,
			documentAsks.partitionsAsked,
			want,
			askedPartition,
		)
	}
}

func TestOnlyAWordWithACompleteAbstractIsCounted(t *testing.T) {
	t.Parallel()

	documentAsks := documentAsksAnswering(map[string]wordpartitionasks.ReplicaAnswer{
		secondWord: completeAbstractIn(askedPartition, 2),
	})
	counts := &countsFromReplicas{}

	amounts := documentAsks.amountsCountedFromReplicas(t, counts)

	if want := amountOfEachWord(map[string]int{secondWord: 2}); !maps.Equal(amounts, want) {
		t.Fatalf(
			"the replicas counted %v, want only the word with a complete abstract",
			amounts,
		)
	}
	want := []documentamounts.PerformedFromReplicas{{
		Partition: askedPartition, AmountOfQueryWords: 2, AmountOfQueryWordsCounted: 1,
	}}
	if !slices.Equal(counts.performed, want) {
		t.Fatalf("the replicas reported %v, want %v", counts.performed, want)
	}
}

func TestWithoutACompleteAbstractNoWordIsCounted(t *testing.T) {
	t.Parallel()

	amounts := documentAsksAnswering(nil).amountsCountedFromReplicas(t, &countsFromReplicas{})

	if len(amounts) != 0 {
		t.Fatalf("the replicas counted %v, want no word without a complete abstract", amounts)
	}
}

func TestACompleteAbstractCountsOnlyTheDocumentsOfThePartitionAsked(t *testing.T) {
	t.Parallel()

	listed := slices.Concat(documentsIn(askedPartition, 1), documentsIn(otherPartition, 4))
	documentAsks := documentAsksAnswering(map[string]wordpartitionasks.ReplicaAnswer{
		firstWord: answerCounting(len(listed), listed...),
	})

	amounts := documentAsks.amountsCountedFromReplicas(t, &countsFromReplicas{})

	if want := amountOfEachWord(map[string]int{firstWord: 1}); !maps.Equal(amounts, want) {
		t.Fatalf("the replicas counted %v, want only the document of the partition asked", amounts)
	}
}

func TestAnAnswerThatListsLessThanItCountsIsNoCompleteAbstract(t *testing.T) {
	t.Parallel()

	documents := documentsIn(askedPartition, 3)
	documentAsks := documentAsksAnswering(map[string]wordpartitionasks.ReplicaAnswer{
		firstWord:  answerCounting(3, documents[0]),
		secondWord: answerCounting(3, documents...),
	})

	amounts := documentAsks.amountsCountedFromReplicas(t, &countsFromReplicas{})

	if want := amountOfEachWord(map[string]int{secondWord: 3}); !maps.Equal(amounts, want) {
		t.Fatalf("the replicas counted %v, want only the second word with 3", amounts)
	}
}

func TestAWordAPeerSearchedAndHoldsNothingForIsCountedWithNoDocument(t *testing.T) {
	t.Parallel()

	documentAsks := documentAsksAnswering(map[string]wordpartitionasks.ReplicaAnswer{
		firstWord:  {Searched: true},
		secondWord: {},
	})

	amounts := documentAsks.amountsCountedFromReplicas(t, &countsFromReplicas{})

	if want := amountOfEachWord(map[string]int{firstWord: 0}); !maps.Equal(amounts, want) {
		t.Fatalf("the replicas counted %v, want only the first word with none", amounts)
	}
}
