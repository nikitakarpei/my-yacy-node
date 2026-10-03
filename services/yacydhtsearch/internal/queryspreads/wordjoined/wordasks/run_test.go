package wordasks_test

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	firstWord              = "berlin"
	secondWord             = "weather"
	twoPartitionsOfTheRing = 2
)

type replicasOfTheNetwork struct {
	answersPerWord map[yacymodel.Hash][]wordpartitionasks.ReplicaAnswer
}

func (replicas replicasOfTheNetwork) Start(context.Context) wordpartitionasks.Run {
	asks := make(chan []wordpartitionasks.Ask)
	settledAsks := make(chan wordpartitionasks.SettledAsk)
	go replicas.settleEach(asks, settledAsks)

	return wordpartitionasks.Run{Asks: asks, SettledAsks: settledAsks}
}

func (replicas replicasOfTheNetwork) settleEach(
	asks <-chan []wordpartitionasks.Ask,
	settledAsks chan<- wordpartitionasks.SettledAsk,
) {
	defer close(settledAsks)
	var unread []wordpartitionasks.SettledAsk
	for asks != nil || len(unread) > 0 {
		var reader chan<- wordpartitionasks.SettledAsk
		var next wordpartitionasks.SettledAsk
		if len(unread) > 0 {
			reader = settledAsks
			next = unread[0]
		}
		select {
		case addedAsks, open := <-asks:
			if !open {
				asks = nil

				continue
			}
			for _, ask := range addedAsks {
				unread = append(unread, wordpartitionasks.SettledAsk{
					Ask:     ask,
					Answers: replicas.answersPerWord[ask.Word],
				})
			}
		case reader <- next:
			unread = unread[1:]
		}
	}
}

func documentsIn(t *testing.T, partition uint, amount int) []yacymodel.URLHash {
	t.Helper()

	partitions := yacymodel.DHTRingPartitions(twoPartitionsOfTheRing)
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

func answerOf(address string, documents ...yacymodel.URLHash) wordpartitionasks.ReplicaAnswer {
	answer := wordpartitionasks.ReplicaAnswer{
		Replica: peerdirectory.AskablePeer{Hash: yacymodel.WordHash(address), Address: address},
	}
	for _, document := range documents {
		answer.ListedDocuments = append(
			answer.ListedDocuments, wordpartitionasks.ListedDocument{Hash: document},
		)
	}

	return answer
}

var query = queryreading.QueryFrom(firstWord+" "+secondWord+" -rain", "de")

func chosenPeersInBothPartitions() peerchoice.ChosenPeersPerQueryWord {
	chosenPeers := []peerchoice.ChosenPeer{
		{Peer: peerdirectory.AskablePeer{Address: "in-0"}, Partition: 0},
		{Peer: peerdirectory.AskablePeer{Address: "in-1"}, Partition: 1},
	}

	return peerchoice.ChosenPeersPerQueryWord{
		{QueryWord: yacymodel.WordHash(firstWord), ChosenPeers: chosenPeers},
		{QueryWord: yacymodel.WordHash(secondWord), ChosenPeers: chosenPeers},
	}
}

func runOver(t *testing.T, replicas replicasOfTheNetwork) *wordasks.Run {
	t.Helper()

	return wordasks.Start(
		t.Context(), replicas, query, chosenPeersInBothPartitions(), twoPartitionsOfTheRing,
	)
}

func wordsOf(spelledWords ...string) []yacymodel.Hash {
	words := make([]yacymodel.Hash, 0, len(spelledWords))
	for _, spelledWord := range spelledWords {
		words = append(words, yacymodel.WordHash(spelledWord))
	}

	return words
}

func TestEveryPartitionOfAWordIsAskedWithTheLanguageAndExclusionsOfTheQuery(t *testing.T) {
	t.Parallel()

	run := runOver(t, replicasOfTheNetwork{})
	run.AskEveryPartitionFor(wordsOf(firstWord))

	answers := run.Finish()

	partitionsAsked := make([]uint, 0, len(answers))
	for _, settledAsk := range answers {
		if settledAsk.Word != yacymodel.WordHash(firstWord) || settledAsk.Language != "de" ||
			!slices.Equal(settledAsk.ExcludedWords, query.ExclusionHashes()) ||
			len(settledAsk.ReplicasInOrder) != 1 {
			t.Fatalf(
				"the ask is %+v, want the word, its language and its exclusions",
				settledAsk.Ask,
			)
		}
		partitionsAsked = append(partitionsAsked, settledAsk.Partition)
	}
	if !slices.Equal(slices.Sorted(slices.Values(partitionsAsked)), []uint{0, 1}) {
		t.Fatalf("the run asked partitions %v, want both", partitionsAsked)
	}
}

func TestTheSettledAsksInAPartitionHoldOnlyThoseOfItsWords(t *testing.T) {
	t.Parallel()

	run := runOver(t, replicasOfTheNetwork{})
	run.AskPartitionFor(1, wordsOf(secondWord))
	run.AskPartitionFor(1, wordsOf(firstWord))
	run.WaitUntilPartitionSettledFor(1, wordsOf(firstWord))

	settledAsks := run.SettledIn(1, wordsOf(firstWord))
	run.Finish()

	if len(settledAsks) != 1 || settledAsks[0].Word != yacymodel.WordHash(firstWord) ||
		settledAsks[0].Partition != 1 {
		t.Fatalf(
			"the settled asks are %v, want the one ask of the first word in partition 1",
			settledAsks,
		)
	}
}

func TestAWordPartitionIsAskedOnceAndTheFirstAskStands(t *testing.T) {
	t.Parallel()

	run := runOver(t, replicasOfTheNetwork{})
	run.AskPartitionFor(0, wordsOf(firstWord))
	askedInZero := run.PartitionAskedFor(0, wordsOf(firstWord))
	askedInOne := run.PartitionAskedFor(1, wordsOf(firstWord))
	run.AskPartitionForWordsAmong(0, wordsOf(firstWord), documentsIn(t, 0, 1))

	answers := run.Finish()

	if !askedInZero || askedInOne || len(answers) != 1 || len(answers[0].DocumentsToMatch) != 0 {
		t.Fatalf(
			"the run asked partition 0 %t and partition 1 %t and settled %v, "+
				"want only partition 0 asked once, whole",
			askedInZero, askedInOne, answers,
		)
	}
}

func TestAnAskAmongDocumentsNamesTheDocumentsToMatch(t *testing.T) {
	t.Parallel()

	documentsToMatch := documentsIn(t, 1, 2)
	run := runOver(t, replicasOfTheNetwork{})
	run.AskPartitionForWordsAmong(1, wordsOf(secondWord), documentsToMatch)

	answers := run.Finish()

	if len(answers) != 1 || !slices.Equal(answers[0].DocumentsToMatch, documentsToMatch) {
		t.Fatalf("the run settled %v, want one ask naming %v", answers, documentsToMatch)
	}
}

func TestAPartitionSettlesAfterTheRunWaitedForItsAsk(t *testing.T) {
	t.Parallel()

	run := runOver(t, replicasOfTheNetwork{})
	run.AskPartitionFor(0, wordsOf(firstWord))
	settledBefore := run.PartitionSettledFor(0, wordsOf(firstWord))

	run.WaitUntilAnyPartitionSettles()
	settledAfter := run.PartitionSettledFor(0, wordsOf(firstWord))
	run.Finish()

	if settledBefore || !settledAfter {
		t.Fatalf(
			"the partition settled %t before and %t after the wait, want only after",
			settledBefore, settledAfter,
		)
	}
}

func TestTheAnswersKnowTheHoldersOfTheListedDocuments(t *testing.T) {
	t.Parallel()

	inPartitionZero := documentsIn(t, 0, 2)
	inPartitionOne := documentsIn(t, 1, 1)
	run := runOver(t, replicasOfTheNetwork{
		answersPerWord: map[yacymodel.Hash][]wordpartitionasks.ReplicaAnswer{
			yacymodel.WordHash(firstWord): {
				answerOf("first", inPartitionZero[0], inPartitionZero[1], inPartitionOne[0]),
				answerOf("second", inPartitionZero[1]),
			},
		},
	})
	run.AskEveryPartitionFor(wordsOf(firstWord))
	run.WaitUntilPartitionSettledFor(0, wordsOf(firstWord))
	run.WaitUntilPartitionSettledFor(1, wordsOf(firstWord))

	documentsInPartitionZero := run.DocumentsListedIn(0, wordsOf(firstWord))
	holders := run.Finish().DocumentHolders()
	got := holders.MostHeldFirst(documentsInPartitionZero)

	if want := []yacymodel.URLHash{inPartitionZero[1], inPartitionZero[0]}; !slices.Equal(
		got, want,
	) {
		t.Fatalf("the documents listed in partition 0 are %v, want %v", got, want)
	}
}

func TestTheAnswersKeepWhatTheReplicasCarried(t *testing.T) {
	t.Parallel()

	documents := documentsIn(t, 0, 2)
	withMetadata := answerOf("first", documents...)
	withMetadata.ListedDocuments[0].Metadata = yacymodel.Some(yacymodel.URLMetadata{
		Hash: documents[0],
	})
	withMetadata.ListedDocuments[0].Posting = yacymodel.Some(yacymodel.RWIPosting{Hits: 1})
	withMetadata.AmountOfDocumentsHeld = yacymodel.Some(512)
	run := runOver(t, replicasOfTheNetwork{
		answersPerWord: map[yacymodel.Hash][]wordpartitionasks.ReplicaAnswer{
			yacymodel.WordHash(firstWord):  {withMetadata},
			yacymodel.WordHash(secondWord): {answerOf("second", documents[1]), answerOf("third")},
		},
	})
	run.AskPartitionFor(0, wordsOf(firstWord, secondWord))

	answers := run.Finish()

	joined := yacymodel.URLHashes{documents[0]: {}, documents[1]: {}}
	if got, want := answers.DocumentsWithoutMetadataAmong(joined), (yacymodel.URLHashes{
		documents[1]: {},
	}); !maps.Equal(got, want) {
		t.Fatalf("the documents without metadata are %v, want %v", got, want)
	}
	wantedMatched := []wordasks.MatchedDocument{{
		Replica:  yacymodel.WordHash("first"),
		Word:     yacymodel.WordHash(firstWord),
		Document: documents[0],
		Metadata: yacymodel.URLMetadata{Hash: documents[0]},
		Posting:  yacymodel.Some(yacymodel.RWIPosting{Hits: 1}),
	}}
	if got := answers.DocumentsThePeersMatched(); !reflect.DeepEqual(got, wantedMatched) {
		t.Fatalf("the peers matched %v, want %v", got, wantedMatched)
	}
	amountOfReplicaAnswers := 0
	for _, settledAsk := range answers {
		amountOfReplicaAnswers += len(settledAsk.Answers)
	}
	if got := amountOfReplicaAnswers; got != 3 {
		t.Fatalf("the answers hold %d replica answers, want 3", got)
	}
	performed := wordasks.PerformedFrom(answers)
	want := wordasks.Performed{
		AmountOfPeersWithANonEmptyAbstract:    2,
		AmountOfMatchedDocumentsAcrossAnswers: 1,
		AmountOfMatchedDocumentsWithAPosting:  1,
		AmountOfDocumentsHeldInEachAnswer:     []int{512},
	}
	if performed.AmountOfPeersWithANonEmptyAbstract != want.AmountOfPeersWithANonEmptyAbstract ||
		performed.AmountOfMatchedDocumentsAcrossAnswers != want.AmountOfMatchedDocumentsAcrossAnswers ||
		performed.AmountOfMatchedDocumentsWithAPosting != want.AmountOfMatchedDocumentsWithAPosting ||
		!slices.Equal(
			performed.AmountOfDocumentsHeldInEachAnswer,
			want.AmountOfDocumentsHeldInEachAnswer,
		) {
		t.Fatalf("the word asks reported %+v, want %+v", performed, want)
	}
}
