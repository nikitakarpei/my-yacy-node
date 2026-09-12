package relevance_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/relevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	amountOfRunsOfTheSameAnswers = 50
	weightOfAWeighedAddressScore = 1.0
)

func metadataOf(t *testing.T, address string, title string) yacymodel.URLMetadata {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return yacymodel.URLMetadata{Hash: hash, Address: address, Title: title}
}

func itemCountedForTheWord(
	t *testing.T, address string, word string, hits int,
) peeranswers.AnsweredItem {
	t.Helper()

	return peeranswers.AnsweredItem{
		Metadata: metadataOf(t, address, ""),
		MatchedWords: map[yacymodel.Hash]peeranswers.WordCount{
			yacymodel.WordHash(word): {Hits: hits},
		},
	}
}

func itemCountedForTheWords(
	t *testing.T, address string, hitsPerWord map[string]int,
) peeranswers.AnsweredItem {
	t.Helper()

	matchedWords := make(map[yacymodel.Hash]peeranswers.WordCount, len(hitsPerWord))
	for word, hits := range hitsPerWord {
		matchedWords[yacymodel.WordHash(word)] = peeranswers.WordCount{Hits: hits}
	}

	return peeranswers.AnsweredItem{
		Metadata:     metadataOf(t, address, ""),
		MatchedWords: matchedWords,
	}
}

func itemOfTextWords(
	t *testing.T, address string, word string, hits int, textWords int,
) peeranswers.AnsweredItem {
	t.Helper()

	return peeranswers.AnsweredItem{
		Metadata: metadataOf(t, address, ""),
		MatchedWords: map[yacymodel.Hash]peeranswers.WordCount{
			yacymodel.WordHash(word): {Hits: hits, TextWords: textWords},
		},
	}
}

func itemMatchingTheWords(
	t *testing.T, address string, words ...string,
) peeranswers.AnsweredItem {
	t.Helper()

	return itemTitledMatchingTheWords(t, address, "", words...)
}

func itemTitledMatchingTheWords(
	t *testing.T, address string, title string, words ...string,
) peeranswers.AnsweredItem {
	t.Helper()

	matchedWords := make([]yacymodel.Hash, 0, len(words))
	for _, word := range words {
		matchedWords = append(matchedWords, yacymodel.WordHash(word))
	}

	return peeranswers.AnsweredItem{Metadata: metadataOf(t, address, title)}.
		MatchingTheWords(matchedWords)
}

const addressNoNodeCanRead = "https://berlin weather.example/"

func itemOfTheAddressNoNodeCanReadMatchingTheWords(
	t *testing.T, words ...string,
) peeranswers.AnsweredItem {
	t.Helper()

	item := itemMatchingTheWords(t, "https://unreadable.example/", words...)
	item.Metadata.Address = addressNoNodeCanRead

	return item
}

func answersOf(
	itemsInTheOrderOfEachPeerRanking ...[]peeranswers.AnsweredItem,
) peeranswers.AnsweredQuery {
	return peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachPeerRanking: itemsInTheOrderOfEachPeerRanking,
	}
}

func answersHolding(
	documentsPerWord map[string]int,
	itemsInTheOrderOfEachPeerRanking ...[]peeranswers.AnsweredItem,
) peeranswers.AnsweredQuery {
	documentsHeldPerQueryWord := make(map[yacymodel.Hash]int, len(documentsPerWord))
	for word, documentsHeldForTheWord := range documentsPerWord {
		documentsHeldPerQueryWord[yacymodel.WordHash(word)] = documentsHeldForTheWord
	}

	return peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachPeerRanking: itemsInTheOrderOfEachPeerRanking,
		DocumentsHeldPerQueryWord:        documentsHeldPerQueryWord,
	}
}

func addressesOrderedBy(answers peeranswers.AnsweredQuery) []string {
	return addressesOrderedByTheScoreWeights(relevance.DefaultScoreWeights(), answers)
}

func addressesOrderedByTheScoreWeights(
	scoreWeights relevance.ScoreWeights, answers peeranswers.AnsweredQuery,
) []string {
	orderedItems := relevance.New(scoreWeights).OrderedItemsOf(answers)

	addresses := make([]string, 0, len(orderedItems))
	for _, orderedItem := range orderedItems {
		addresses = append(addresses, orderedItem.Metadata.Address)
	}

	return addresses
}

func TestTheDocumentOfTheRarerQueryWordComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100000, "kelondro": 10},
		[]peeranswers.AnsweredItem{
			itemCountedForTheWord(t, "https://common.example/", "berlin", 1),
		},
		[]peeranswers.AnsweredItem{
			itemCountedForTheWord(t, "https://rare.example/", "kelondro", 1),
		},
	)

	want := []string{"https://rare.example/", "https://common.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestEveryQueryWordWeighsTheSameWhenNoPeerCountedDocumentsForIt(t *testing.T) {
	t.Parallel()

	answers := answersOf(
		[]peeranswers.AnsweredItem{itemCountedForTheWord(t, "https://a.example/", "berlin", 1)},
		[]peeranswers.AnsweredItem{itemCountedForTheWord(t, "https://b.example/", "kelondro", 1)},
	)

	want := []string{"https://a.example/", "https://b.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want the order the peers put %v", got, want)
	}
}

func TestTheWordNoPeerCountedDocumentsForWeighsAsMuchAsTheMostCommonCountedWord(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]peeranswers.AnsweredItem{
			itemCountedForTheWord(t, "https://counted.example/", "berlin", 1),
		},
		[]peeranswers.AnsweredItem{
			itemCountedForTheWord(t, "https://uncounted.example/", "kelondro", 1),
		},
	)

	want := []string{"https://counted.example/", "https://uncounted.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want the order the peers put %v", got, want)
	}
}

func TestTheDocumentWithMoreHitsOfTheSameWordComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]peeranswers.AnsweredItem{itemCountedForTheWord(t, "https://once.example/", "berlin", 1)},
		[]peeranswers.AnsweredItem{
			itemCountedForTheWord(t, "https://often.example/", "berlin", 9),
		},
	)

	want := []string{"https://often.example/", "https://once.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestEachFurtherHitOfTheSameWordAddsLessThanTheFirstHitOfAnotherWord(t *testing.T) {
	t.Parallel()

	twiceOneWord := itemCountedForTheWord(t, "https://one-word.example/", "berlin", 2)
	onceEachWord := peeranswers.AnsweredItem{
		Metadata: metadataOf(t, "https://two-words.example/", ""),
		MatchedWords: map[yacymodel.Hash]peeranswers.WordCount{
			yacymodel.WordHash("berlin"):  {Hits: 1},
			yacymodel.WordHash("weather"): {Hits: 1},
		},
	}
	answers := answersOf(
		[]peeranswers.AnsweredItem{twiceOneWord},
		[]peeranswers.AnsweredItem{onceEachWord},
	)

	want := []string{"https://two-words.example/", "https://one-word.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheShorterDocumentOfTheSameHitsComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]peeranswers.AnsweredItem{itemOfTextWords(t, "https://long.example/", "berlin", 3, 5000)},
		[]peeranswers.AnsweredItem{itemOfTextWords(t, "https://short.example/", "berlin", 3, 100)},
	)

	want := []string{"https://short.example/", "https://long.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentThatMatchedMoreQueryWordsComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100, "weather": 100},
		[]peeranswers.AnsweredItem{
			itemCountedForTheWords(t, "https://one.example/", map[string]int{"berlin": 1}),
		},
		[]peeranswers.AnsweredItem{
			itemCountedForTheWords(t, "https://both.example/",
				map[string]int{"berlin": 1, "weather": 1}),
		},
	)

	want := []string{"https://both.example/", "https://one.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentOfEveryQueryWordComesBeforeOneOfManyHitsOfASingleQueryWord(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100, "weather": 100},
		[]peeranswers.AnsweredItem{
			itemCountedForTheWords(t, "https://one-word.example/",
				map[string]int{"berlin": 50, "weather": 0}),
		},
		[]peeranswers.AnsweredItem{
			itemCountedForTheWords(t, "https://every-word.example/",
				map[string]int{"berlin": 1, "weather": 1}),
		},
	)

	want := []string{"https://every-word.example/", "https://one-word.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentWhoseHostHoldsTheOnlyQueryWordComesBeforeOneOfHitsOfThatWord(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]peeranswers.AnsweredItem{
			itemCountedForTheWords(t, "https://berlin.example/", map[string]int{"berlin": 0}),
		},
		[]peeranswers.AnsweredItem{
			itemCountedForTheWords(t, "https://weather.example/", map[string]int{"berlin": 5}),
		},
	)

	want := []string{"https://berlin.example/", "https://weather.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentNoOneCountedAHitInComesAfterOneWithAHit(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100, "weather": 100},
		[]peeranswers.AnsweredItem{
			itemMatchingTheWords(t, "https://uncounted.example/", "berlin", "weather"),
		},
		[]peeranswers.AnsweredItem{
			itemCountedForTheWords(t, "https://counted.example/", map[string]int{"berlin": 1}),
		},
	)

	want := []string{"https://counted.example/", "https://uncounted.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentWithoutATitleComesAfterOneAPeerPutBehindIt(t *testing.T) {
	t.Parallel()

	answers := answersOf(
		[]peeranswers.AnsweredItem{
			itemMatchingTheWords(t, "https://untitled.example/", "berlin"),
			itemTitledMatchingTheWords(t, "https://titled.example/", "A city", "berlin"),
		},
	)

	want := []string{"https://titled.example/", "https://untitled.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentWhoseTitleHoldsTheQueryWordComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]peeranswers.AnsweredItem{
			itemTitledMatchingTheWords(t, "https://beside.example/", "The weather of a city",
				"berlin"),
		},
		[]peeranswers.AnsweredItem{
			itemTitledMatchingTheWords(t, "https://titled.example/", "Berlin weather", "berlin"),
		},
	)

	want := []string{"https://titled.example/", "https://beside.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentWhoseTitleHoldsTheRarerQueryWordComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"emacs": 10, "manual": 100000},
		[]peeranswers.AnsweredItem{
			itemTitledMatchingTheWords(t, "https://beside.example/", "The manual",
				"emacs", "manual"),
			itemTitledMatchingTheWords(t, "https://titled.example/", "The emacs",
				"emacs", "manual"),
		},
	)

	want := []string{"https://titled.example/", "https://beside.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestATitleThatOnlyHoldsALongerWordChangesNoOrder(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"tofu": 100},
		[]peeranswers.AnsweredItem{
			itemTitledMatchingTheWords(t, "https://beside.example/", "Migrating to a fork",
				"tofu"),
		},
		[]peeranswers.AnsweredItem{
			itemTitledMatchingTheWords(t, "https://titled.example/", "Migrating to OpenTofu",
				"tofu"),
		},
	)

	want := []string{"https://beside.example/", "https://titled.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestATitleIsReadPastItsPunctuation(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"opentofu": 100},
		[]peeranswers.AnsweredItem{itemMatchingTheWords(t, "https://beside.example/", "opentofu")},
		[]peeranswers.AnsweredItem{
			itemTitledMatchingTheWords(t, "https://titled.example/",
				"Terraform vs. OpenTofu: what changed?", "opentofu"),
		},
	)

	want := []string{"https://titled.example/", "https://beside.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestAWeighedAddressScorePutsTheDocumentWhoseHostHoldsTheQueryWordFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]peeranswers.AnsweredItem{
			itemMatchingTheWords(t, "https://weather.example/city/", "berlin"),
		},
		[]peeranswers.AnsweredItem{
			itemMatchingTheWords(t, "https://berlin.example/city/", "berlin"),
		},
	)
	scoreWeights := relevance.DefaultScoreWeights()
	scoreWeights.WeightOfTheAddressScore = weightOfAWeighedAddressScore

	want := []string{"https://berlin.example/city/", "https://weather.example/city/"}
	got := addressesOrderedByTheScoreWeights(scoreWeights, answers)
	if !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

const hitsOfTheQueryWordInEveryPhrasedDocument = 3

func itemHoldingTheQueryPhrase(
	t *testing.T, address string, queryPhraseHits int,
) peeranswers.AnsweredItem {
	t.Helper()

	item := itemCountedForTheWord(
		t, address, "berlin", hitsOfTheQueryWordInEveryPhrasedDocument,
	)
	item.QueryPhraseHits = queryPhraseHits

	return item
}

func TestTheDocumentWhoseTextHoldsTheQueryPhraseComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]peeranswers.AnsweredItem{
			itemHoldingTheQueryPhrase(t, "https://apart.example/", 0),
		},
		[]peeranswers.AnsweredItem{
			itemHoldingTheQueryPhrase(t, "https://phrased.example/", 1),
		},
	)

	want := []string{"https://phrased.example/", "https://apart.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestEachFurtherQueryPhraseHitAddsLessThanTheFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]peeranswers.AnsweredItem{
			itemHoldingTheQueryPhrase(t, "https://once.example/", 1),
			itemHoldingTheQueryPhrase(t, "https://often.example/", 9),
		},
	)

	want := []string{"https://often.example/", "https://once.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentOfAnAddressNoNodeCanReadKeepsThePlaceThePeersPutIt(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]peeranswers.AnsweredItem{itemOfTheAddressNoNodeCanReadMatchingTheWords(t, "berlin")},
		[]peeranswers.AnsweredItem{
			itemMatchingTheWords(t, "https://weather.example/city/", "berlin"),
		},
	)

	want := []string{addressNoNodeCanRead, "https://weather.example/city/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want the order the peers put %v", got, want)
	}
}

func TestTheDocumentTwoPeersPutHighComesBeforeOneASinglePeerPutFirst(t *testing.T) {
	t.Parallel()

	putByBothPeers := itemCountedForTheWord(t, "https://both.example/", "berlin", 1)
	answers := answersOf(
		[]peeranswers.AnsweredItem{
			itemCountedForTheWord(t, "https://one.example/", "berlin", 1), putByBothPeers,
		},
		[]peeranswers.AnsweredItem{putByBothPeers},
	)

	want := []string{"https://both.example/", "https://one.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestAnItemOfNoOrderComesAfterAnEquallyCountedItemAPeerPut(t *testing.T) {
	t.Parallel()

	answers := peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachPeerRanking: [][]peeranswers.AnsweredItem{
			{itemCountedForTheWord(t, "https://ordered.example/", "berlin", 1)},
		},
		ItemsInNoOrder: []peeranswers.AnsweredItem{
			itemCountedForTheWord(t, "https://unordered.example/", "berlin", 1),
		},
	}

	want := []string{"https://ordered.example/", "https://unordered.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheSameAnswersComeBackInTheSameOrderEveryRun(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100000, "weather": 7000, "kelondro": 13, "freeworld": 421},
		[]peeranswers.AnsweredItem{
			itemCountedForTheWords(t, "https://a.example/",
				map[string]int{"berlin": 3, "weather": 2, "kelondro": 1, "freeworld": 4}),
			itemCountedForTheWords(t, "https://b.example/",
				map[string]int{"berlin": 4, "weather": 1, "kelondro": 1, "freeworld": 3}),
			itemCountedForTheWords(t, "https://c.example/",
				map[string]int{"berlin": 2, "weather": 3, "kelondro": 2, "freeworld": 1}),
		},
		[]peeranswers.AnsweredItem{
			itemCountedForTheWords(t, "https://d.example/",
				map[string]int{"berlin": 1, "weather": 4, "kelondro": 1, "freeworld": 2}),
			itemCountedForTheWords(t, "https://e.example/",
				map[string]int{"berlin": 3, "weather": 3, "kelondro": 1, "freeworld": 1}),
		},
	)

	want := addressesOrderedBy(answers)
	for run := range amountOfRunsOfTheSameAnswers {
		if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
			t.Fatalf("run %d reads the relevance order %v, want %v", run, got, want)
		}
	}
}

func TestDocumentsOfTheSameRelevanceKeepTheOrderThePeersPutThem(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]peeranswers.AnsweredItem{itemCountedForTheWord(t, "https://a.example/", "berlin", 1)},
		[]peeranswers.AnsweredItem{itemCountedForTheWord(t, "https://b.example/", "berlin", 1)},
	)

	want := []string{"https://a.example/", "https://b.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want the order the peers put %v", got, want)
	}
}
