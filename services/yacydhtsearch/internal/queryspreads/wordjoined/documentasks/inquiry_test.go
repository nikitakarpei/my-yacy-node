package documentasks_test

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
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	firstWord               = "berlin"
	secondWord              = "weather"
	twoPartitionsOfTheRing  = 2
	documentsToMatchCeiling = 2
)

type replicasOfTheNetwork struct {
	answersPerWord            map[yacymodel.Hash][]wordpartitionasks.ReplicaAnswer
	settledAsksReadAtEachAsk  *[]int
	asksInTheOrderTheyWerePut *[]wordpartitionasks.Ask
}

func replicasAnswering(
	answersPerWord map[yacymodel.Hash][]wordpartitionasks.ReplicaAnswer,
) replicasOfTheNetwork {
	return replicasOfTheNetwork{
		answersPerWord:            answersPerWord,
		settledAsksReadAtEachAsk:  &[]int{},
		asksInTheOrderTheyWerePut: &[]wordpartitionasks.Ask{},
	}
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
	settledAsksRead := 0
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
				*replicas.settledAsksReadAtEachAsk = append(
					*replicas.settledAsksReadAtEachAsk, settledAsksRead,
				)
				*replicas.asksInTheOrderTheyWerePut = append(
					*replicas.asksInTheOrderTheyWerePut,
					ask,
				)
				unread = append(unread, wordpartitionasks.SettledAsk{
					Ask:     ask,
					Answers: replicas.answersPerWord[ask.Word],
				})
			}
		case reader <- next:
			unread = unread[1:]
			settledAsksRead++
		}
	}
}

func (replicas replicasOfTheNetwork) asksOf(spelledWord string) []wordpartitionasks.Ask {
	var asksOfTheWord []wordpartitionasks.Ask
	for _, ask := range *replicas.asksInTheOrderTheyWerePut {
		if ask.Word == yacymodel.WordHash(spelledWord) {
			asksOfTheWord = append(asksOfTheWord, ask)
		}
	}

	return asksOfTheWord
}

func (replicas replicasOfTheNetwork) settledAsksReadAtTheAsksOf(spelledWord string) []int {
	var settledAsksRead []int
	for place, ask := range *replicas.asksInTheOrderTheyWerePut {
		if ask.Word == yacymodel.WordHash(spelledWord) {
			settledAsksRead = append(settledAsksRead, (*replicas.settledAsksReadAtEachAsk)[place])
		}
	}

	return settledAsksRead
}

type askedPartitions struct {
	decisionPerPartition map[uint]documentasks.DocumentsToMatchDecision
}

func (asked *askedPartitions) AskedAmongTheDocuments(
	_ context.Context,
	decisionPerPartition documentasks.DocumentsToMatchDecisionPerPartition,
) {
	maps.Copy(asked.decisionPerPartition, decisionPerPartition)
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

func firstWordListing(documents ...yacymodel.URLHash) replicasOfTheNetwork {
	return replicasAnswering(map[yacymodel.Hash][]wordpartitionasks.ReplicaAnswer{
		yacymodel.WordHash(firstWord): {answerOf("first", documents...)},
	})
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

func inquiryOver(
	t *testing.T,
	replicas replicasOfTheNetwork,
	observer documentasks.DocumentAsksObserver,
) *documentasks.Inquiry {
	t.Helper()

	return documentasks.New(replicas, twoPartitionsOfTheRing, documentsToMatchCeiling, observer).
		Begin(t.Context(), query, chosenPeersInBothPartitions())
}

func wordsOf(spelledWords ...string) []yacymodel.Hash {
	words := make([]yacymodel.Hash, 0, len(spelledWords))
	for _, spelledWord := range spelledWords {
		words = append(words, yacymodel.WordHash(spelledWord))
	}

	return words
}

func askedFirstThenSecondAmongIt(
	t *testing.T,
	replicas replicasOfTheNetwork,
	amountOfTheFirstInAPartition int,
) map[uint]documentasks.DocumentsToMatchDecision {
	t.Helper()

	asked := &askedPartitions{
		decisionPerPartition: map[uint]documentasks.DocumentsToMatchDecision{},
	}
	inquiry := inquiryOver(t, replicas, asked)
	inquiry.ExpectInAPartition(yacymodel.WordHash(firstWord), amountOfTheFirstInAPartition)
	inquiry.WhichDocumentsHavingTheseAlsoHave(wordsOf(firstWord), wordsOf(secondWord))
	inquiry.End()

	return asked.decisionPerPartition
}

func TestEveryPartitionOfAWordIsAskedWithTheLanguageAndExclusionsOfTheQuery(t *testing.T) {
	t.Parallel()

	inquiry := inquiryOver(t, replicasAnswering(nil), documentasks.DocumentAsksObservers{})
	answers := inquiry.WhichDocumentsHave(wordsOf(firstWord))

	inquiry.End()

	partitionsAsked := make([]uint, 0, len(answers.SettledAsks))
	for _, settledAsk := range answers.SettledAsks {
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
		t.Fatalf("the inquiry asked partitions %v, want both", partitionsAsked)
	}
}

func TestTheSettledAsksInAPartitionHoldOnlyThoseOfItsWords(t *testing.T) {
	t.Parallel()

	inquiry := inquiryOver(t, replicasAnswering(nil), documentasks.DocumentAsksObservers{})
	inquiry.WhichDocumentsHaveIn(1, wordsOf(secondWord))

	settledAsks := inquiry.WhichDocumentsHaveIn(1, wordsOf(firstWord)).SettledAsks
	inquiry.End()

	if len(settledAsks) != 1 || settledAsks[0].Word != yacymodel.WordHash(firstWord) ||
		settledAsks[0].Partition != 1 {
		t.Fatalf(
			"the settled asks are %v, want the one ask of the first word in partition 1",
			settledAsks,
		)
	}
}

func TestAWordPartitionIsAskedOnce(t *testing.T) {
	t.Parallel()

	replicas := replicasAnswering(nil)
	inquiry := inquiryOver(t, replicas, documentasks.DocumentAsksObservers{})
	inquiry.WhichDocumentsHaveIn(0, wordsOf(firstWord))
	inquiry.WhichDocumentsHave(wordsOf(firstWord))

	inquiry.End()

	partitionsAsked := make([]uint, 0, len(*replicas.asksInTheOrderTheyWerePut))
	for _, ask := range *replicas.asksInTheOrderTheyWerePut {
		partitionsAsked = append(partitionsAsked, ask.Partition)
	}
	if !slices.Equal(partitionsAsked, []uint{0, 1}) {
		t.Fatalf(
			"the inquiry asked partitions %v, want partition 0 then partition 1",
			partitionsAsked,
		)
	}
}

func TestEachPartitionIsAskedByWhatItsDocumentsToMatchAllow(t *testing.T) {
	t.Parallel()

	documentsToMatch := documentsIn(t, 1, 1)
	replicas := firstWordListing(documentsToMatch...)

	got := askedFirstThenSecondAmongIt(t, replicas, 0)

	want := map[uint]documentasks.DocumentsToMatchDecision{
		0: documentasks.NoDocumentsToMatch,
		1: documentasks.NamedTheDocumentsToMatch,
	}
	if !maps.Equal(got, want) {
		t.Fatalf("the partitions were asked %v, want %v", got, want)
	}
	asksOfTheSecondWord := replicas.asksOf(secondWord)
	if len(asksOfTheSecondWord) != 1 || asksOfTheSecondWord[0].Partition != 1 ||
		!slices.Equal(asksOfTheSecondWord[0].DocumentsToMatch, documentsToMatch) {
		t.Fatalf(
			"the inquiry asked %v, want one ask in partition 1 naming %v",
			asksOfTheSecondWord, documentsToMatch,
		)
	}
}

func TestTheDocumentsToMatchAreNamedInHashOrder(t *testing.T) {
	t.Parallel()

	documents := documentsIn(t, 1, 2)
	replicas := firstWordListing(documents[1], documents[0])

	askedFirstThenSecondAmongIt(t, replicas, 0)

	want := yacymodel.URLHashes{documents[0]: {}, documents[1]: {}}.InHashOrder()
	if asksOfTheSecondWord := replicas.asksOf(secondWord); len(asksOfTheSecondWord) != 1 ||
		!slices.Equal(asksOfTheSecondWord[0].DocumentsToMatch, want) {
		t.Fatalf("the inquiry asked %v, want one ask naming %v", asksOfTheSecondWord, want)
	}
}

func TestAPartitionWithMoreDocumentsToMatchThanTheCeilingIsAskedWhole(t *testing.T) {
	t.Parallel()

	replicas := firstWordListing(append(documentsIn(t, 0, 3), documentsIn(t, 1, 1)...)...)

	got := askedFirstThenSecondAmongIt(t, replicas, 0)

	want := map[uint]documentasks.DocumentsToMatchDecision{
		0: documentasks.NamedNoneOverTheCeiling,
		1: documentasks.NamedTheDocumentsToMatch,
	}
	asksOfTheSecondWord := replicas.asksOf(secondWord)
	asksInPartitionZero := slices.DeleteFunc(
		slices.Clone(asksOfTheSecondWord),
		func(ask wordpartitionasks.Ask) bool { return ask.Partition != 0 },
	)
	if !maps.Equal(got, want) || len(asksInPartitionZero) != 1 ||
		len(asksInPartitionZero[0].DocumentsToMatch) != 0 {
		t.Fatalf("the partitions were asked %v by %v, want %v", got, asksOfTheSecondWord, want)
	}
}

func TestAPartitionWhoseWordsToAskWereAskedAlreadyReportsNoKind(t *testing.T) {
	t.Parallel()

	asked := &askedPartitions{
		decisionPerPartition: map[uint]documentasks.DocumentsToMatchDecision{},
	}
	inquiry := inquiryOver(t, firstWordListing(documentsIn(t, 1, 1)...), asked)
	inquiry.WhichDocumentsHaveIn(0, wordsOf(secondWord))

	inquiry.WhichDocumentsHavingTheseAlsoHave(wordsOf(firstWord), wordsOf(secondWord))
	inquiry.End()

	if want := (map[uint]documentasks.DocumentsToMatchDecision{1: documentasks.NamedTheDocumentsToMatch}); !maps.Equal(
		asked.decisionPerPartition,
		want,
	) {
		t.Fatalf("the partitions were asked %v, want %v", asked.decisionPerPartition, want)
	}
}

func TestDocumentsOverTheCeilingInAPartitionHaveTheWordsToAskAskedAtTheStart(t *testing.T) {
	t.Parallel()

	replicas := firstWordListing(documentsIn(t, 1, 1)...)

	got := askedFirstThenSecondAmongIt(t, replicas, documentsToMatchCeiling+1)

	want := map[uint]documentasks.DocumentsToMatchDecision{
		0: documentasks.NamedNonePredictedOverTheCeiling,
		1: documentasks.NamedNonePredictedOverTheCeiling,
	}
	if read := replicas.settledAsksReadAtTheAsksOf(secondWord); !maps.Equal(got, want) ||
		!slices.Equal(read, []int{0, 0}) {
		t.Fatalf(
			"the partitions were asked %v after %v settled asks were read, "+
				"want %v before any was read",
			got, read, want,
		)
	}
}

func TestDocumentsAtTheCeilingInAPartitionHaveTheWordsToAskWaitForThePartition(t *testing.T) {
	t.Parallel()

	replicas := firstWordListing(documentsIn(t, 1, 1)...)

	askedFirstThenSecondAmongIt(t, replicas, documentsToMatchCeiling)

	if read := replicas.settledAsksReadAtTheAsksOf(secondWord); len(read) != 1 || read[0] == 0 {
		t.Fatalf(
			"the word to ask was asked after %v settled asks were read, "+
				"want one ask after its partition settled",
			read,
		)
	}
}

func TestTheAnswersKnowTheHoldersOfTheListedDocuments(t *testing.T) {
	t.Parallel()

	documents := documentsIn(t, 0, 2)
	inquiry := inquiryOver(
		t,
		replicasAnswering(map[yacymodel.Hash][]wordpartitionasks.ReplicaAnswer{
			yacymodel.WordHash(firstWord): {
				answerOf("first", documents[0], documents[1]),
				answerOf("second", documents[1]),
			},
		}),
		documentasks.DocumentAsksObservers{},
	)
	answers := inquiry.WhichDocumentsHaveIn(0, wordsOf(firstWord))
	inquiry.End()

	holders := answers.HoldersOf(yacymodel.URLHashes{documents[0]: {}, documents[1]: {}})
	got := holders.MostHeldFirst()

	if want := []yacymodel.URLHash{documents[1], documents[0]}; !slices.Equal(got, want) {
		t.Fatalf("the documents most held first are %v, want %v", got, want)
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
	inquiry := inquiryOver(
		t,
		replicasAnswering(map[yacymodel.Hash][]wordpartitionasks.ReplicaAnswer{
			yacymodel.WordHash(firstWord):  {withMetadata},
			yacymodel.WordHash(secondWord): {answerOf("second", documents[1]), answerOf("third")},
		}),
		documentasks.DocumentAsksObservers{},
	)
	answers := inquiry.WhichDocumentsHaveIn(0, wordsOf(firstWord, secondWord))
	inquiry.End()

	joined := yacymodel.URLHashes{documents[0]: {}, documents[1]: {}}
	if got, want := answers.DocumentsWithoutMetadataAmong(joined), (yacymodel.URLHashes{
		documents[1]: {},
	}); !maps.Equal(got, want) {
		t.Fatalf("the documents without metadata are %v, want %v", got, want)
	}
	wantedListed := []documentasks.ListedDocumentWithMetadata{{
		Replica:  yacymodel.WordHash("first"),
		Word:     yacymodel.WordHash(firstWord),
		Document: documents[0],
		Metadata: yacymodel.URLMetadata{Hash: documents[0]},
		Posting:  yacymodel.Some(yacymodel.RWIPosting{Hits: 1}),
	}}
	if got := answers.ListedDocumentsWithMetadata(); !reflect.DeepEqual(got, wantedListed) {
		t.Fatalf("the peers listed with metadata %v, want %v", got, wantedListed)
	}
	performed := documentasks.PerformedFrom(answers)
	want := documentasks.Performed{
		AmountOfPeersWithANonEmptyAbstract:  2,
		AmountOfListedDocumentsWithMetadata: 1,
		AmountOfListedDocumentsWithAPosting: 1,
		AmountOfDocumentsHeldInEachAnswer:   []int{512},
	}
	if !reflect.DeepEqual(performed, want) {
		t.Fatalf("the document asks reported %+v, want %+v", performed, want)
	}
}
