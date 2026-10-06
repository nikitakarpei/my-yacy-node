package documentamounts_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentamounts"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	firstWord              = "berlin"
	secondWord             = "weather"
	twoPartitionsOfTheRing = 2
	askedPartition         = 1
)

var query = queryreading.QueryFrom(firstWord+" "+secondWord, "")

type documentAsksOfTheNetwork struct {
	amountInTheCompleteAbstractOfEachWord map[string]int
	partitionsAsked                       []uint
}

func documentAsksAnswering(
	amountInTheCompleteAbstractOfEachWord map[string]int,
) *documentAsksOfTheNetwork {
	return &documentAsksOfTheNetwork{
		amountInTheCompleteAbstractOfEachWord: amountInTheCompleteAbstractOfEachWord,
	}
}

func (documentAsks *documentAsksOfTheNetwork) WhichDocumentsHaveIn(
	partition uint,
	words []yacymodel.Hash,
) []documentasks.AnsweredWordPartition {
	documentAsks.partitionsAsked = append(documentAsks.partitionsAsked, partition)
	var answered []documentasks.AnsweredWordPartition
	for spelledWord, amount := range documentAsks.amountInTheCompleteAbstractOfEachWord {
		if !slices.Contains(words, yacymodel.WordHash(spelledWord)) {
			continue
		}
		answered = append(answered, documentasks.AnsweredWordPartition{
			Word:      yacymodel.WordHash(spelledWord),
			Partition: partition,
			Answers: []wordpartitionasks.ReplicaAnswer{{
				ListedDocuments:       documentsIn(partition, amount),
				AmountOfDocumentsHeld: yacymodel.Some(amount),
			}},
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

func documentsIn(partition uint, amount int) []wordpartitionasks.ListedDocument {
	partitions := yacymodel.DHTRingPartitions(twoPartitionsOfTheRing)
	documents := make([]wordpartitionasks.ListedDocument, 0, amount)
	for place := 0; len(documents) < amount; place++ {
		document, err := yacymodel.URLHashOf(fmt.Sprintf("https://document-%d.example/", place))
		if err != nil || partitions.PartitionOf(document) != partition {
			continue
		}
		documents = append(documents, wordpartitionasks.ListedDocument{Hash: document})
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

	documentAsks := documentAsksAnswering(map[string]int{firstWord: 3, secondWord: 1})

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

	documentAsks := documentAsksAnswering(map[string]int{secondWord: 2})
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
