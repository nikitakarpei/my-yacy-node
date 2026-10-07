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

func (*askedPartitions) DocumentAsksPerformed(context.Context, documentasks.Performed) {}

type documentAsksReports struct {
	performed []documentasks.Performed
}

func (*documentAsksReports) AskedAmongTheDocuments(
	context.Context,
	documentasks.DocumentsToMatchDecisionPerPartition,
) {
}

func (reports *documentAsksReports) DocumentAsksPerformed(
	_ context.Context,
	performed documentasks.Performed,
) {
	reports.performed = append(reports.performed, performed)
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

var query = queryreading.QueryFrom(firstWord+" "+secondWord+" -rain", languageFrom("de"))

func languageFrom(code string) yacymodel.Language {
	language, err := yacymodel.ParseLanguage(code)
	if err != nil {
		panic(err)
	}

	return language
}

func chosenPeersInBothPartitions() peerchoice.ChosenPeersPerQueryWord {
	chosenPeers := []peerchoice.ChosenPeer{
		{Peer: peerdirectory.AskablePeer{Address: "in-0"}, Partition: 0},
		{Peer: peerdirectory.AskablePeer{Address: "in-1"}, Partition: 1},
	}

	return peerchoice.ChosenPeersPerQueryWord{
		{QueryWord: yacymodel.WordHash(firstWord), ChosenPeers: chosenPeers},
		{QueryWord: yacymodel.WordHash(secondWord), ChosenPeers: chosenPeers},
		{QueryWord: query.CompoundWords[0].Hash(), ChosenPeers: chosenPeers},
	}
}

type answeredWordPartition struct {
	word      yacymodel.Hash
	partition uint
	answers   []wordpartitionasks.ReplicaAnswer
}

type wordPartitionsAnsweredInTurn struct {
	answered []answeredWordPartition
}

func (inquirer *wordPartitionsAnsweredInTurn) WordPartitionAnswered(
	word yacymodel.Hash,
	partition uint,
	answers []wordpartitionasks.ReplicaAnswer,
) {
	inquirer.answered = append(
		inquirer.answered,
		answeredWordPartition{word: word, partition: partition, answers: answers},
	)
}

func (inquirer *wordPartitionsAnsweredInTurn) wordPartitions() []string {
	wordPartitions := make([]string, 0, len(inquirer.answered))
	for _, answered := range inquirer.answered {
		wordPartitions = append(wordPartitions, wordPartitionOf(answered.word, answered.partition))
	}

	return wordPartitions
}

func wordPartitionOf(word yacymodel.Hash, partition uint) string {
	return fmt.Sprintf("%s in %d", word, partition)
}

func wordPartitionsOf(spelledWord string, partitions ...uint) []string {
	wordPartitions := make([]string, 0, len(partitions))
	for _, partition := range partitions {
		wordPartitions = append(
			wordPartitions, wordPartitionOf(yacymodel.WordHash(spelledWord), partition),
		)
	}

	return wordPartitions
}

func inquiryOver(
	t *testing.T,
	replicas replicasOfTheNetwork,
	observer documentasks.DocumentAsksObserver,
	inquirer documentasks.Inquirer,
) *documentasks.Inquiry {
	t.Helper()

	return documentasks.New(replicas, twoPartitionsOfTheRing, documentsToMatchCeiling, observer).
		Begin(t.Context(), query, chosenPeersInBothPartitions(), inquirer)
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
	inquiry := inquiryOver(t, replicas, asked, &wordPartitionsAnsweredInTurn{})
	inquiry.ExpectInAPartition(yacymodel.WordHash(firstWord), amountOfTheFirstInAPartition)
	inquiry.WhichDocumentsHavingTheseAlsoHave(wordsOf(firstWord), wordsOf(secondWord))
	inquiry.End()

	return asked.decisionPerPartition
}

func TestEveryInquirerIsToldEachAnsweredWordPartition(t *testing.T) {
	t.Parallel()

	first, second := &wordPartitionsAnsweredInTurn{}, &wordPartitionsAnsweredInTurn{}
	inquiry := inquiryOver(
		t,
		firstWordListing(),
		documentasks.DocumentAsksObservers{},
		documentasks.Inquirers{first, second},
	)

	inquiry.WhichDocumentsHave(wordsOf(firstWord))
	inquiry.End()

	if len(first.wordPartitions()) == 0 ||
		!reflect.DeepEqual(first.wordPartitions(), second.wordPartitions()) {
		t.Fatalf(
			"the inquirers were told %v and %v, want both told every answered word partition",
			first.wordPartitions(), second.wordPartitions(),
		)
	}
}

func TestEveryPartitionOfAWordIsAskedWithTheLanguageAndExclusionsOfTheQuery(t *testing.T) {
	t.Parallel()

	replicas := replicasAnswering(nil)
	inquiry := inquiryOver(
		t, replicas, documentasks.DocumentAsksObservers{}, &wordPartitionsAnsweredInTurn{},
	)
	inquiry.WhichDocumentsHave(wordsOf(firstWord))

	inquiry.End()

	asksOfTheWord := replicas.asksOf(firstWord)
	partitionsAsked := make([]uint, 0, len(asksOfTheWord))
	for _, ask := range asksOfTheWord {
		if ask.Language.String() != "de" ||
			!slices.Equal(ask.ExcludedWords, query.ExclusionHashes()) ||
			len(ask.ReplicasInOrder) != 1 {
			t.Fatalf("the ask is %+v, want the word, its language and its exclusions", ask)
		}
		partitionsAsked = append(partitionsAsked, ask.Partition)
	}
	if !slices.Equal(slices.Sorted(slices.Values(partitionsAsked)), []uint{0, 1}) {
		t.Fatalf("the inquiry asked partitions %v, want both", partitionsAsked)
	}
}

func TestEverySettledAskReachesTheInquirerOnceWithItsWordAndPartition(t *testing.T) {
	t.Parallel()

	inquirer := &wordPartitionsAnsweredInTurn{}
	inquiry := inquiryOver(
		t,
		replicasAnswering(nil),
		documentasks.DocumentAsksObservers{},
		inquirer,
	)
	inquiry.WhichDocumentsHave(wordsOf(secondWord))
	inquiry.WhichDocumentsHave(wordsOf(firstWord, secondWord))

	inquiry.End()

	want := slices.Sorted(slices.Values(
		slices.Concat(wordPartitionsOf(secondWord, 0, 1), wordPartitionsOf(firstWord, 0, 1)),
	))
	if got := slices.Sorted(slices.Values(inquirer.wordPartitions())); !slices.Equal(got, want) {
		t.Fatalf("the inquirer got %v, want %v", got, want)
	}
}

func TestTheAsksOfAQuestionReachTheInquirerBeforeItReturns(t *testing.T) {
	t.Parallel()

	documents := documentsIn(t, 1, 1)
	inquirer := &wordPartitionsAnsweredInTurn{}
	inquiry := inquiryOver(
		t, firstWordListing(documents...), documentasks.DocumentAsksObservers{}, inquirer,
	)

	inquiry.WhichDocumentsHave(wordsOf(firstWord, secondWord))
	answeredBeforeTheEnd := slices.Sorted(slices.Values(inquirer.wordPartitions()))
	inquiry.End()

	want := slices.Sorted(slices.Values(
		slices.Concat(wordPartitionsOf(firstWord, 0, 1), wordPartitionsOf(secondWord, 0, 1)),
	))
	if !slices.Equal(answeredBeforeTheEnd, want) {
		t.Fatalf("the inquirer got %v before the end, want %v", answeredBeforeTheEnd, want)
	}
}

func TestEveryAskOfAQuestionAmongTheDocumentsReachesTheInquirerBeforeItReturns(t *testing.T) {
	t.Parallel()

	inquirer := &wordPartitionsAnsweredInTurn{}
	inquiry := inquiryOver(
		t,
		firstWordListing(documentsIn(t, 1, 1)...),
		&askedPartitions{decisionPerPartition: map[uint]documentasks.DocumentsToMatchDecision{}},
		inquirer,
	)

	inquiry.WhichDocumentsHavingTheseAlsoHave(wordsOf(firstWord), wordsOf(secondWord))
	answeredBeforeTheEnd := slices.Sorted(slices.Values(inquirer.wordPartitions()))
	inquiry.End()

	want := slices.Sorted(slices.Values(
		slices.Concat(wordPartitionsOf(firstWord, 0, 1), wordPartitionsOf(secondWord, 1)),
	))
	if !slices.Equal(answeredBeforeTheEnd, want) || len(inquirer.answered) != len(want) {
		t.Fatalf(
			"the inquirer got %v before the end and %d in all, want %v once each",
			answeredBeforeTheEnd, len(inquirer.answered), want,
		)
	}
}

func TestAWordPartitionIsAskedOnce(t *testing.T) {
	t.Parallel()

	replicas := replicasAnswering(nil)
	inquiry := inquiryOver(
		t, replicas, documentasks.DocumentAsksObservers{}, &wordPartitionsAnsweredInTurn{},
	)
	inquiry.WhichDocumentsHave(wordsOf(firstWord))
	inquiry.WhichDocumentsHave(wordsOf(firstWord))

	inquiry.End()

	partitionsAsked := make([]uint, 0, len(*replicas.asksInTheOrderTheyWerePut))
	for _, ask := range *replicas.asksInTheOrderTheyWerePut {
		partitionsAsked = append(partitionsAsked, ask.Partition)
	}
	if !slices.Equal(slices.Sorted(slices.Values(partitionsAsked)), []uint{0, 1}) {
		t.Fatalf("the inquiry asked partitions %v, want each partition once", partitionsAsked)
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

func TestTheInquiryReportsWhatTheReplicasCarriedOnceItEnds(t *testing.T) {
	t.Parallel()

	documents := documentsIn(t, 0, 2)
	withMetadata := answerOf("first", documents...)
	withMetadata.ListedDocuments[0] = wordpartitionasks.ListedDocumentFrom(
		yacymodel.URLMetadata{Hash: documents[0]},
		yacymodel.Some(yacymodel.RWIPosting{Hits: 1}),
	)
	withMetadata.AmountOfDocumentsHeld = yacymodel.Some(512)
	reports := &documentAsksReports{}
	inquiry := inquiryOver(
		t,
		replicasAnswering(map[yacymodel.Hash][]wordpartitionasks.ReplicaAnswer{
			yacymodel.WordHash(firstWord):  {withMetadata},
			yacymodel.WordHash(secondWord): {answerOf("second", documents[1]), answerOf("third")},
		}),
		reports,
		&wordPartitionsAnsweredInTurn{},
	)
	inquiry.WhichDocumentsHave(wordsOf(firstWord, secondWord))
	amountOfReportsBeforeTheEnd := len(reports.performed)
	inquiry.End()

	want := []documentasks.Performed{{
		AmountOfPeersWithANonEmptyAbstract:  2,
		AmountOfListedDocumentsWithMetadata: twoPartitionsOfTheRing,
		AmountOfListedDocumentsWithAPosting: twoPartitionsOfTheRing,
		AmountOfDocumentsHeldInEachAnswer:   []int{512, 512},
	}}
	if amountOfReportsBeforeTheEnd != 0 || !reflect.DeepEqual(reports.performed, want) {
		t.Fatalf(
			"the document asks reported %d times before the end and %+v in all, "+
				"want one report at the end: %+v",
			amountOfReportsBeforeTheEnd, reports.performed, want,
		)
	}
}

func TestTheReportCountsTheCompoundWordsThePartitionsAnswered(t *testing.T) {
	t.Parallel()

	reports := &documentAsksReports{}
	inquiry := inquiryOver(
		t, replicasAnswering(nil), reports, &wordPartitionsAnsweredInTurn{},
	)
	inquiry.WhichDocumentsHave(query.HashesOfWordsAndCompoundWords())
	inquiry.End()

	if len(reports.performed) != 1 || reports.performed[0].AmountOfCompoundWordsAnswered != 1 {
		t.Fatalf(
			"the document asks reported %+v for the compound words %v, want the one answered",
			reports.performed, query.CompoundWords,
		)
	}
}
