package documentamounts_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
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

type replicaAnswerOfTheNetwork struct {
	amountListed  int
	amountCounted int
}

type documentAsksOfTheNetwork struct {
	answerOfEachWord map[string]replicaAnswerOfTheNetwork
	partitionsAsked  []uint
}

func (documentAsks *documentAsksOfTheNetwork) WhichDocumentsHaveIn(
	partition uint,
	words []yacymodel.Hash,
) documentasks.Answers {
	documentAsks.partitionsAsked = append(documentAsks.partitionsAsked, partition)
	var settledAsks []wordpartitionasks.SettledAsk
	for spelledWord, answer := range documentAsks.answerOfEachWord {
		if !slices.Contains(words, yacymodel.WordHash(spelledWord)) {
			continue
		}
		settledAsks = append(settledAsks, wordpartitionasks.SettledAsk{
			Ask: wordpartitionasks.Ask{Word: yacymodel.WordHash(spelledWord), Partition: partition},
			Answers: []wordpartitionasks.ReplicaAnswer{{
				Replica:               peerdirectory.AskablePeer{Address: spelledWord},
				ListedDocuments:       documentsIn(partition, answer.amountListed),
				AmountOfDocumentsHeld: yacymodel.Some(answer.amountCounted),
			}},
		})
	}

	return documentasks.Answers{SettledAsks: settledAsks, Partitions: twoPartitionsOfTheRing}
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

	documentAsks := &documentAsksOfTheNetwork{
		answerOfEachWord: map[string]replicaAnswerOfTheNetwork{
			firstWord:  {amountListed: 3, amountCounted: 3},
			secondWord: {amountListed: 1, amountCounted: 1},
		},
	}

	amounts := amountsFromReplicas(&countsFromReplicas{}).
		AmountsInAPartitionFor(t.Context(), query, documentAsks)

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

	documentAsks := &documentAsksOfTheNetwork{
		answerOfEachWord: map[string]replicaAnswerOfTheNetwork{
			firstWord:  {amountListed: 1, amountCounted: 4},
			secondWord: {amountListed: 2, amountCounted: 2},
		},
	}
	counts := &countsFromReplicas{}

	amounts := amountsFromReplicas(counts).AmountsInAPartitionFor(t.Context(), query, documentAsks)

	if want := amountOfEachWord(map[string]int{secondWord: 2}); !maps.Equal(amounts, want) {
		t.Fatalf(
			"the replicas counted %v, want only the word whose answer lists all it counts",
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

func TestWithoutAnAnswerNoWordIsCounted(t *testing.T) {
	t.Parallel()

	amounts := amountsFromReplicas(&countsFromReplicas{}).
		AmountsInAPartitionFor(t.Context(), query, &documentAsksOfTheNetwork{})

	if len(amounts) != 0 {
		t.Fatalf("the replicas counted %v, want no word where no replica answered", amounts)
	}
}
