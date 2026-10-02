package wordholdings_test

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordholdings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	firstWord              = "berlin"
	secondWord             = "weather"
	twoPartitionsOfTheRing = 2
)

var query = queryreading.QueryFrom(firstWord+" "+secondWord, "")

type replicaAnswer struct {
	documents []yacymodel.URLHash
	counted   yacymodel.Optional[int]
	searched  bool
}

func countedAnswer(amountHeld int, documents ...yacymodel.URLHash) replicaAnswer {
	return replicaAnswer{documents: documents, counted: yacymodel.Some(amountHeld)}
}

func settledAsk(
	spelledWord string,
	partition uint,
	answers ...replicaAnswer,
) wordpartitionasks.SettledAsk {
	settled := wordpartitionasks.SettledAsk{
		Ask: wordpartitionasks.Ask{Word: yacymodel.WordHash(spelledWord), Partition: partition},
	}
	for place, answer := range answers {
		replicaAnswer := wordpartitionasks.ReplicaAnswer{
			Replica: peerdirectory.AskablePeer{
				Hash:    yacymodel.WordHash(fmt.Sprintf("replica-%d", place)),
				Address: fmt.Sprintf("replica-%d", place),
			},
			AmountOfDocumentsHeld: answer.counted,
			Searched:              answer.searched,
		}
		for _, document := range answer.documents {
			replicaAnswer.ListedDocuments = append(
				replicaAnswer.ListedDocuments, wordpartitionasks.ListedDocument{Hash: document},
			)
		}
		settled.Answers = append(settled.Answers, replicaAnswer)
	}

	return settled
}

func documentsInPartition(
	t *testing.T,
	partitions yacymodel.DHTRingPartitions,
	partition uint,
	amount int,
) []yacymodel.URLHash {
	t.Helper()

	documents := make([]yacymodel.URLHash, 0, amount)
	for place := 0; len(documents) < amount; place++ {
		document, err := yacymodel.URLHashOf(fmt.Sprintf("https://document-%d.example/", place))
		if err != nil {
			t.Fatalf("document %d has no hash: %v", place, err)
		}
		if partitions.PartitionOf(document) != partition {
			continue
		}
		documents = append(documents, document)
	}

	return documents
}

func heldForBothWords(amountHeld int) map[yacymodel.Hash]int {
	return map[yacymodel.Hash]int{
		yacymodel.WordHash(firstWord):  amountHeld,
		yacymodel.WordHash(secondWord): amountHeld,
	}
}

func asksOfBothWords(partition uint, answers ...replicaAnswer) []wordpartitionasks.SettledAsk {
	return []wordpartitionasks.SettledAsk{
		settledAsk(firstWord, partition, answers...),
		settledAsk(secondWord, partition, answers...),
	}
}

func TestEveryPartitionOfAQueryWordAddsWhatItsReplicasCounted(t *testing.T) {
	t.Parallel()

	settledAsks := slices.Concat(
		asksOfBothWords(0, countedAnswer(100), countedAnswer(100), countedAnswer(100)),
		asksOfBothWords(1, countedAnswer(10)),
	)

	got := wordholdings.OfEachQueryWord(query, settledAsks, twoPartitionsOfTheRing).
		AmountOfDocumentsHeldPerQueryWord()

	if want := heldForBothWords(110); !maps.Equal(got, want) {
		t.Fatalf("the holdings estimate %v, want %v", got, want)
	}
}

func TestAPartitionOfAnEvenAmountOfCountsTakesTheLowerMiddleOne(t *testing.T) {
	t.Parallel()

	settledAsks := asksOfBothWords(0, countedAnswer(10), countedAnswer(20))

	got := wordholdings.OfEachQueryWord(query, settledAsks, 1).AmountOfDocumentsHeldPerQueryWord()

	if want := heldForBothWords(10); !maps.Equal(got, want) {
		t.Fatalf("the holdings estimate %v, want %v", got, want)
	}
}

func TestAPartitionNoPeerCountedTakesTheMiddleOfThePartitionsThatWereCounted(t *testing.T) {
	t.Parallel()

	settledAsks := slices.Concat(
		asksOfBothWords(0, countedAnswer(10)),
		asksOfBothWords(1, countedAnswer(30)),
		asksOfBothWords(2, replicaAnswer{}),
	)

	got := wordholdings.OfEachQueryWord(query, settledAsks, 3).AmountOfDocumentsHeldPerQueryWord()

	if want := heldForBothWords(10 + 30 + 10); !maps.Equal(got, want) {
		t.Fatalf("the holdings estimate %v, want %v", got, want)
	}
}

func TestAQueryWordNoPeerCountedCarriesNoDocumentsHeld(t *testing.T) {
	t.Parallel()

	settledAsks := slices.Concat(
		asksOfBothWords(0, replicaAnswer{}),
		asksOfBothWords(1, replicaAnswer{}),
	)

	got := wordholdings.OfEachQueryWord(query, settledAsks, twoPartitionsOfTheRing).
		AmountOfDocumentsHeldPerQueryWord()

	if len(got) != 0 {
		t.Fatalf("the holdings estimate %v, want none", got)
	}
}

func TestTheAmountInAPartitionIsTheMiddleOfThePartitionsThatWereCounted(t *testing.T) {
	t.Parallel()

	settledAsks := slices.Concat(
		asksOfBothWords(0, countedAnswer(10)),
		asksOfBothWords(1, countedAnswer(30), countedAnswer(50)),
		asksOfBothWords(2, countedAnswer(20)),
		asksOfBothWords(3, replicaAnswer{}),
	)

	got := wordholdings.OfEachQueryWord(query, settledAsks, 4).
		AmountOfDocumentsInAPartitionPerQueryWord()

	if want := heldForBothWords(20); !maps.Equal(got, want) {
		t.Fatalf("the holdings give %v in a partition, want %v", got, want)
	}
}

func TestAQueryWordNoPeerCountedHasNoAmountInAPartition(t *testing.T) {
	t.Parallel()

	settledAsks := []wordpartitionasks.SettledAsk{
		settledAsk(firstWord, 0, replicaAnswer{}),
		settledAsk(secondWord, 0, countedAnswer(10)),
	}

	got := wordholdings.OfEachQueryWord(query, settledAsks, twoPartitionsOfTheRing).
		AmountOfDocumentsInAPartitionPerQueryWord()

	if want := (map[yacymodel.Hash]int{yacymodel.WordHash(secondWord): 10}); !maps.Equal(
		got, want,
	) {
		t.Fatalf("the holdings give %v in a partition, want %v", got, want)
	}
}

func TestOnlyTheDocumentsEveryQueryWordHoldsAreJoined(t *testing.T) {
	t.Parallel()

	documents := documentsInPartition(t, 1, 0, 3)
	settledAsks := []wordpartitionasks.SettledAsk{
		settledAsk(firstWord, 0, countedAnswer(2, documents[0], documents[1])),
		settledAsk(secondWord, 0, countedAnswer(2, documents[0], documents[2])),
	}

	got := wordholdings.OfEachQueryWord(query, settledAsks, 1).DocumentsOfEveryWord()

	if want := (yacymodel.URLHashes{documents[0]: {}}); !maps.Equal(got, want) {
		t.Fatalf("the join holds %v, want %v", got, want)
	}
}

func TestADocumentInTheAbstractOfTheCompoundWordIsJoinedForBothItsParts(t *testing.T) {
	t.Parallel()

	documents := documentsInPartition(t, 1, 0, 2)
	settledAsks := []wordpartitionasks.SettledAsk{
		settledAsk(firstWord, 0, countedAnswer(1, documents[0])),
		settledAsk(firstWord+secondWord, 0, countedAnswer(1, documents[1])),
	}

	holdings := wordholdings.OfEachQueryWord(query, settledAsks, 1)

	if got, want := holdings.DocumentsOfEveryWord(), (yacymodel.URLHashes{documents[1]: {}}); !maps.Equal(
		got,
		want,
	) {
		t.Fatalf("the join holds %v, want the document of the compound word %v", got, want)
	}
	if holdings.AmountOfCompoundWords() != 1 || holdings.AmountOfQueryWords() != 2 ||
		holdings.AmountOfQueryWordsHeldByNoPeer() != 1 {
		t.Fatalf(
			"the holdings count %d compound words, %d query words, %d held by no peer, "+
				"want 1, 2 and 1",
			holdings.AmountOfCompoundWords(),
			holdings.AmountOfQueryWords(),
			holdings.AmountOfQueryWordsHeldByNoPeer(),
		)
	}
}

func amountOfDocumentsInEach(
	completeAbstracts []wordholdings.CompleteAbstract,
) map[yacymodel.Hash]int {
	amountOfDocumentsPerWord := map[yacymodel.Hash]int{}
	for _, completeAbstract := range completeAbstracts {
		amountOfDocumentsPerWord[completeAbstract.Word] = len(
			completeAbstract.DocumentsInThePartition,
		)
	}

	return amountOfDocumentsPerWord
}

func TestACompleteAbstractHoldsOnlyTheDocumentsOfItsPartition(t *testing.T) {
	t.Parallel()

	inPartitionZero := documentsInPartition(t, twoPartitionsOfTheRing, 0, 2)
	inPartitionOne := documentsInPartition(t, twoPartitionsOfTheRing, 1, 4)
	settledAsks := []wordpartitionasks.SettledAsk{
		settledAsk(
			firstWord, 0, countedAnswer(5, append(inPartitionZero[:1:1], inPartitionOne...)...),
		),
		settledAsk(secondWord, 0, countedAnswer(2, inPartitionZero...)),
	}

	got := wordholdings.OfEachQueryWord(query, settledAsks, twoPartitionsOfTheRing).
		CompleteAbstractsIn(0)

	want := map[yacymodel.Hash]int{
		yacymodel.WordHash(firstWord): 1, yacymodel.WordHash(secondWord): 2,
	}
	if !maps.Equal(amountOfDocumentsInEach(got), want) ||
		got[0].Word != yacymodel.WordHash(firstWord) {
		t.Fatalf("the complete abstracts are %v, want %v in query order", got, want)
	}
}

func TestAnAnswerThatListsLessThanItCountsIsNoCompleteAbstract(t *testing.T) {
	t.Parallel()

	documents := documentsInPartition(t, 1, 0, 3)
	settledAsks := []wordpartitionasks.SettledAsk{
		settledAsk(firstWord, 0, countedAnswer(3, documents[0])),
		settledAsk(secondWord, 0, countedAnswer(3, documents...)),
	}

	got := wordholdings.OfEachQueryWord(query, settledAsks, 1).CompleteAbstractsIn(0)

	if want := map[yacymodel.Hash]int{yacymodel.WordHash(secondWord): 3}; !maps.Equal(
		amountOfDocumentsInEach(got), want,
	) {
		t.Fatalf("the complete abstracts are %v, want only the second word with 3", got)
	}
}

func TestAWordAPeerSearchedAndHoldsNothingForHasAnEmptyCompleteAbstract(t *testing.T) {
	t.Parallel()

	settledAsks := []wordpartitionasks.SettledAsk{
		settledAsk(firstWord, 0, replicaAnswer{searched: true}),
		settledAsk(secondWord, 0, replicaAnswer{}),
	}

	got := wordholdings.OfEachQueryWord(query, settledAsks, 1).CompleteAbstractsIn(0)

	if want := map[yacymodel.Hash]int{yacymodel.WordHash(firstWord): 0}; !maps.Equal(
		amountOfDocumentsInEach(got), want,
	) {
		t.Fatalf("the complete abstracts are %v, want only the first word with none", got)
	}
}

func TestWithoutAChosenWordTheWordWithFewestDocumentsLeads(t *testing.T) {
	t.Parallel()

	documents := documentsInPartition(t, 1, 0, 3)
	settledAsks := []wordpartitionasks.SettledAsk{
		settledAsk(firstWord, 0, countedAnswer(3, documents...)),
		settledAsk(secondWord, 0, countedAnswer(1, documents[0])),
	}
	holdings := wordholdings.OfEachQueryWord(query, settledAsks, 1)

	if got := holdings.DocumentsOfTheLeadingWord(yacymodel.None[yacymodel.Hash]()); len(got) != 1 {
		t.Fatalf("the leading word holds %v, want the one document of the rarer word", got)
	}
	if got := holdings.DocumentsOfTheLeadingWord(
		yacymodel.Some(yacymodel.WordHash(firstWord)),
	); len(got) != 3 {
		t.Fatalf("the leading word holds %v, want the three documents of the chosen word", got)
	}
}
