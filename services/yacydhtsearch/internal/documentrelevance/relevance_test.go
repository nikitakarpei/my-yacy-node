package documentrelevance_test

import (
	"cmp"
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	amountOfRunsOfTheSameAnswers   = 50
	amountOfWordsOfALinkedDocument = 1000

	hitsOfTheQueryWordInEveryUnreadDocument     = 3
	amountOfWordsAPeerCountedOfAnUnreadDocument = 1200
)

type answeredDocument struct {
	foundDocument queryanswers.FoundDocument
	facts         queryanswers.DocumentFacts
}

func foundDocumentAt(t *testing.T, address string) answeredDocument {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return answeredDocument{
		foundDocument: queryanswers.FoundDocument{Hash: hash, Address: address},
		facts:         queryanswers.DocumentFacts{HitsPerQueryWord: map[yacymodel.Hash]int{}},
	}
}

func foundDocumentWithHitsOf(
	t *testing.T, address string, word string, hits int,
) answeredDocument {
	t.Helper()

	return foundDocumentWithHitsPerWord(t, address, map[string]int{word: hits})
}

func foundDocumentWithHitsPerWord(
	t *testing.T, address string, hitsPerWord map[string]int,
) answeredDocument {
	t.Helper()

	foundDocument := foundDocumentAt(t, address)
	for word, hits := range hitsPerWord {
		foundDocument.facts.HitsPerQueryWord[yacymodel.WordHash(word)] = hits
	}

	return foundDocument
}

func foundDocumentWithHitsAmongWords(
	t *testing.T, address string, word string, hits int, amountOfWords int,
) answeredDocument {
	t.Helper()

	foundDocument := foundDocumentWithHitsOf(t, address, word, hits)
	foundDocument.facts.AmountOfWords = yacymodel.Some(amountOfWords)

	return foundDocument
}

func foundDocumentMatchingTheWords(
	t *testing.T, address string, words ...string,
) answeredDocument {
	t.Helper()

	return foundDocumentTitledMatchingTheWords(t, address, "", words...)
}

func foundDocumentTitledMatchingTheWords(
	t *testing.T, address string, title string, words ...string,
) answeredDocument {
	t.Helper()

	foundDocument := foundDocumentAt(t, address)
	foundDocument.foundDocument.Title = title
	for _, word := range words {
		foundDocument.facts.HitsPerQueryWord[yacymodel.WordHash(word)] = 0
	}

	return foundDocument
}

const addressNoNodeCanRead = "https://berlin weather.example/"

func foundDocumentOfTheAddressNoNodeCanReadMatchingTheWords(
	t *testing.T, words ...string,
) answeredDocument {
	t.Helper()

	foundDocument := foundDocumentMatchingTheWords(t, "https://unreadable.example/", words...)
	foundDocument.foundDocument.Address = addressNoNodeCanRead

	return foundDocument
}

func answersOf(
	queryWords []string,
	answeredDocuments []answeredDocument,
) queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords:       hashesOfTheQueryWords(queryWords),
		FoundDocuments:   foundDocumentsOf(answeredDocuments),
		FactsPerDocument: factsPerDocumentOf(answeredDocuments),
	}
}

func foundDocumentsOf(answeredDocuments []answeredDocument) []queryanswers.FoundDocument {
	foundDocuments := make([]queryanswers.FoundDocument, 0, len(answeredDocuments))
	for _, answered := range answeredDocuments {
		foundDocuments = append(foundDocuments, answered.foundDocument)
	}

	return foundDocuments
}

func factsPerDocumentOf(answeredDocuments []answeredDocument) queryanswers.FactsPerDocument {
	factsPerDocument := make(queryanswers.FactsPerDocument, len(answeredDocuments))
	for _, answered := range answeredDocuments {
		factsPerDocument[answered.foundDocument.Hash] = answered.facts
	}

	return factsPerDocument
}

func answersHolding(
	documentsPerWord map[string]int,
	queryWords []string,
	answeredDocuments []answeredDocument,
) queryanswers.AnsweredQuery {
	documentsHeldPerQueryWord := make(map[yacymodel.Hash]int, len(documentsPerWord))
	for word, documentsHeldForTheWord := range documentsPerWord {
		documentsHeldPerQueryWord[yacymodel.WordHash(word)] = documentsHeldForTheWord
	}

	return queryanswers.AnsweredQuery{
		QueryWords:                hashesOfTheQueryWords(queryWords),
		FoundDocuments:            foundDocumentsOf(answeredDocuments),
		FactsPerDocument:          factsPerDocumentOf(answeredDocuments),
		DocumentsHeldPerQueryWord: documentsHeldPerQueryWord,
	}
}

func hashesOfTheQueryWords(queryWords []string) []yacymodel.Hash {
	hashes := make([]yacymodel.Hash, 0, len(queryWords))
	for _, queryWord := range queryWords {
		hashes = append(hashes, yacymodel.WordHash(queryWord))
	}

	return hashes
}

func addressesInFallingOrderOfRelevance(answers queryanswers.AnsweredQuery) []string {
	return addressesInFallingOrderOfRelevanceByTheScoreWeights(
		documentrelevance.DefaultScoreWeights(), answers,
	)
}

func addressesInFallingOrderOfRelevanceByTheScoreWeights(
	scoreWeights documentrelevance.ScoreWeights, answers queryanswers.AnsweredQuery,
) []string {
	relevancePerDocument := documentrelevance.New(scoreWeights).RelevancePerDocumentOf(answers)
	foundDocuments := slices.Clone(answers.FoundDocuments)
	slices.SortStableFunc(foundDocuments, func(one, other queryanswers.FoundDocument) int {
		return cmp.Compare(
			relevancePerDocument[other.Hash], relevancePerDocument[one.Hash],
		)
	})

	addresses := make([]string, 0, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		addresses = append(addresses, foundDocument.Address)
	}

	return addresses
}

func TestTheDocumentOfTheRarerQueryWordComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100000, "kelondro": 10},
		[]string{"berlin", "kelondro"},
		[]answeredDocument{
			foundDocumentWithHitsOf(t, "https://common.example/", "berlin", 1),
			foundDocumentWithHitsOf(t, "https://rare.example/", "kelondro", 1),
		},
	)

	want := []string{"https://rare.example/", "https://common.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestEveryQueryWordWeighsTheSameWhenNoPeerCountedDocumentsForIt(t *testing.T) {
	t.Parallel()

	answers := answersOf(
		[]string{"berlin", "kelondro"},
		[]answeredDocument{
			foundDocumentWithHitsOf(t, "https://a.example/", "berlin", 1),
			foundDocumentWithHitsOf(t, "https://b.example/", "kelondro", 1),
		},
	)

	want := []string{"https://a.example/", "https://b.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want the found order %v", got, want)
	}
}

func TestTheWordNoPeerCountedDocumentsForWeighsAsMuchAsTheMostCommonCountedWord(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]string{"berlin", "kelondro"},
		[]answeredDocument{
			foundDocumentWithHitsOf(t, "https://counted.example/", "berlin", 1),
			foundDocumentWithHitsOf(t, "https://uncounted.example/", "kelondro", 1),
		},
	)

	want := []string{"https://counted.example/", "https://uncounted.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want the found order %v", got, want)
	}
}

func TestTheDocumentWithMoreHitsOfTheSameWordComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]string{"berlin"},
		[]answeredDocument{
			foundDocumentWithHitsOf(t, "https://once.example/", "berlin", 1),
			foundDocumentWithHitsOf(t, "https://often.example/", "berlin", 9),
		},
	)

	want := []string{"https://often.example/", "https://once.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestEachFurtherHitOfTheSameWordAddsLessThanTheFirstHitOfAnotherWord(t *testing.T) {
	t.Parallel()

	twiceOneWord := foundDocumentWithHitsOf(t, "https://one-word.example/", "berlin", 2)
	onceEachWord := foundDocumentWithHitsPerWord(
		t, "https://two-words.example/", map[string]int{"berlin": 1, "weather": 1},
	)
	answers := answersOf(
		[]string{"berlin", "weather"},
		[]answeredDocument{
			twiceOneWord,
			onceEachWord,
		},
	)

	want := []string{"https://two-words.example/", "https://one-word.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheShorterDocumentOfTheSameHitsComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]string{"berlin"},
		[]answeredDocument{
			foundDocumentWithHitsAmongWords(t, "https://long.example/", "berlin", 3, 5000),
			foundDocumentWithHitsAmongWords(t, "https://short.example/", "berlin", 3, 100),
		},
	)

	want := []string{"https://short.example/", "https://long.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentThatMatchedMoreQueryWordsComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100, "weather": 100},
		[]string{"berlin", "weather"},
		[]answeredDocument{
			foundDocumentWithHitsPerWord(t, "https://one.example/", map[string]int{"berlin": 1}),
			foundDocumentWithHitsPerWord(t, "https://both.example/",
				map[string]int{"berlin": 1, "weather": 1}),
		},
	)

	want := []string{"https://both.example/", "https://one.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentWhoseHostHoldsTheOnlyQueryWordComesBeforeOneOfHitsOfThatWord(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]string{"berlin"},
		[]answeredDocument{
			foundDocumentWithHitsPerWord(t, "https://berlin.example/", map[string]int{"berlin": 0}),
			foundDocumentWithHitsPerWord(
				t,
				"https://weather.example/",
				map[string]int{"berlin": 5},
			),
		},
	)

	want := []string{"https://berlin.example/", "https://weather.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentNoOneCountedAHitInComesAfterOneWithAHit(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100, "weather": 100},
		[]string{"berlin", "weather"},
		[]answeredDocument{
			foundDocumentMatchingTheWords(t, "https://uncounted.example/", "berlin", "weather"),
			foundDocumentWithHitsPerWord(
				t,
				"https://counted.example/",
				map[string]int{"berlin": 1},
			),
		},
	)

	want := []string{"https://counted.example/", "https://uncounted.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentWithoutATitleComesAfterOneAPeerPutBehindIt(t *testing.T) {
	t.Parallel()

	answers := answersOf(
		[]string{"berlin"},
		[]answeredDocument{
			foundDocumentMatchingTheWords(t, "https://untitled.example/", "berlin"),
			foundDocumentTitledMatchingTheWords(t, "https://titled.example/", "A city", "berlin"),
		},
	)

	want := []string{"https://titled.example/", "https://untitled.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentWhoseTitleHoldsTheQueryWordComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]string{"berlin"},
		[]answeredDocument{
			foundDocumentTitledMatchingTheWords(
				t,
				"https://beside.example/",
				"The weather of a city",
				"berlin",
			),
			foundDocumentTitledMatchingTheWords(
				t,
				"https://titled.example/",
				"Berlin weather",
				"berlin",
			),
		},
	)

	want := []string{"https://titled.example/", "https://beside.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentWhoseTitleHoldsTheRarerQueryWordComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"emacs": 10, "manual": 100000},
		[]string{"emacs", "manual"},
		[]answeredDocument{
			foundDocumentTitledMatchingTheWords(t, "https://beside.example/", "The manual",
				"emacs", "manual"),
			foundDocumentTitledMatchingTheWords(t, "https://titled.example/", "The emacs",
				"emacs", "manual"),
		},
	)

	want := []string{"https://titled.example/", "https://beside.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestATitleThatOnlyHoldsALongerWordChangesNoOrder(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"tofu": 100},
		[]string{"tofu"},
		[]answeredDocument{
			foundDocumentTitledMatchingTheWords(t, "https://beside.example/", "Migrating to a fork",
				"tofu"),
			foundDocumentTitledMatchingTheWords(
				t,
				"https://titled.example/",
				"Migrating to OpenTofu",
				"tofu",
			),
		},
	)

	want := []string{"https://beside.example/", "https://titled.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestATitleIsReadPastItsPunctuation(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"opentofu": 100},
		[]string{"opentofu"},
		[]answeredDocument{
			foundDocumentMatchingTheWords(t, "https://beside.example/", "opentofu"),
			foundDocumentTitledMatchingTheWords(t, "https://titled.example/",
				"Terraform vs. OpenTofu: what changed?", "opentofu"),
		},
	)

	want := []string{"https://titled.example/", "https://beside.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

const hitsOfTheQueryWordInEveryPhrasedDocument = 3

func foundDocumentHoldingTheQueryPhrase(
	t *testing.T, address string, queryPhraseHits int,
) answeredDocument {
	t.Helper()

	foundDocument := foundDocumentWithHitsOf(
		t, address, "berlin", hitsOfTheQueryWordInEveryPhrasedDocument,
	)
	foundDocument.facts.QueryPhraseHits = yacymodel.Some(queryPhraseHits)

	return foundDocument
}

func TestTheDocumentWhoseTextHoldsTheQueryPhraseComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]string{"berlin"},
		[]answeredDocument{
			foundDocumentHoldingTheQueryPhrase(t, "https://apart.example/", 0),
			foundDocumentHoldingTheQueryPhrase(t, "https://phrased.example/", 1),
		},
	)

	want := []string{"https://phrased.example/", "https://apart.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestEachFurtherQueryPhraseHitAddsLessThanTheFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]string{"berlin"},
		[]answeredDocument{
			foundDocumentHoldingTheQueryPhrase(t, "https://once.example/", 1),
			foundDocumentHoldingTheQueryPhrase(t, "https://often.example/", 9),
		},
	)

	want := []string{"https://often.example/", "https://once.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentOfAnAddressNoNodeCanReadKeepsThePlaceItWasFoundAt(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]string{"berlin"},
		[]answeredDocument{
			foundDocumentOfTheAddressNoNodeCanReadMatchingTheWords(t, "berlin"),
			foundDocumentMatchingTheWords(t, "https://weather.example/city/", "berlin"),
		},
	)

	want := []string{addressNoNodeCanRead, "https://weather.example/city/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want the found order %v", got, want)
	}
}

func TestTheSameAnswersComeBackInTheSameOrderEveryRun(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100000, "weather": 7000, "kelondro": 13, "freeworld": 421},
		[]string{"berlin", "weather", "kelondro", "freeworld"},
		[]answeredDocument{
			foundDocumentWithHitsPerWord(t, "https://a.example/",
				map[string]int{"berlin": 3, "weather": 2, "kelondro": 1, "freeworld": 4}),
			foundDocumentWithHitsPerWord(t, "https://b.example/",
				map[string]int{"berlin": 4, "weather": 1, "kelondro": 1, "freeworld": 3}),
			foundDocumentWithHitsPerWord(t, "https://c.example/",
				map[string]int{"berlin": 2, "weather": 3, "kelondro": 2, "freeworld": 1}),
			foundDocumentWithHitsPerWord(t, "https://d.example/",
				map[string]int{"berlin": 1, "weather": 4, "kelondro": 1, "freeworld": 2}),
			foundDocumentWithHitsPerWord(t, "https://e.example/",
				map[string]int{"berlin": 3, "weather": 3, "kelondro": 1, "freeworld": 1}),
		},
	)

	want := addressesInFallingOrderOfRelevance(answers)
	for run := range amountOfRunsOfTheSameAnswers {
		if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
			t.Fatalf("run %d reads the relevance order %v, want %v", run, got, want)
		}
	}
}

func TestTwoDocumentsOfTheSameCountedHitsHoldTheSameRelevance(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]string{"berlin"},
		[]answeredDocument{
			foundDocumentWithHitsOf(t, "https://a.example/", "berlin", 1),
			foundDocumentWithHitsOf(t, "https://b.example/", "berlin", 1),
		},
	)

	relevancePerDocument := documentrelevance.New(documentrelevance.DefaultScoreWeights()).
		RelevancePerDocumentOf(answers)
	relevanceOfTheDocuments := slices.Collect(maps.Values(relevancePerDocument))
	if len(relevanceOfTheDocuments) != 2 ||
		relevanceOfTheDocuments[0] != relevanceOfTheDocuments[1] {
		t.Fatalf(
			"the relevance per document reads %v, want the same relevance for both documents",
			relevancePerDocument,
		)
	}
}

func TestTheEntryPageOfTheSiteTheQueryNamesComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"heise": 100},
		[]string{"heise"},
		[]answeredDocument{
			foundDocumentMatchingTheWords(t, "https://www.heise.de/developer/kontakt/", "heise"),
			foundDocumentMatchingTheWords(t, "https://heise-academy.de/", "heise"),
			foundDocumentMatchingTheWords(t, "https://www.heise.de/", "heise"),
		},
	)

	want := []string{
		"https://www.heise.de/",
		"https://heise-academy.de/",
		"https://www.heise.de/developer/kontakt/",
	}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func foundDocumentWithLinksAmongAThousandWords(
	t *testing.T, address string, amountOfLinks int,
) answeredDocument {
	t.Helper()

	foundDocument := foundDocumentWithHitsOf(t, address, "berlin", 1)
	foundDocument.facts.AmountOfWords = yacymodel.Some(amountOfWordsOfALinkedDocument)
	foundDocument.facts.AmountOfLinks = yacymodel.Some(amountOfLinks)

	return foundDocument
}

func TestTheDocumentOfFewerLinksPerWordComesLast(t *testing.T) {
	t.Parallel()

	answers := answersOf(
		[]string{"berlin"},
		[]answeredDocument{
			foundDocumentWithLinksAmongAThousandWords(t, "https://sparse.example/", 1),
			foundDocumentWithLinksAmongAThousandWords(t, "https://linked.example/", 50),
		},
	)

	want := []string{"https://linked.example/", "https://sparse.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentNoPeerSentLinkCountsForComesBetweenTheSparseAndTheLinked(t *testing.T) {
	t.Parallel()

	withoutLinkCounts := foundDocumentWithLinksAmongAThousandWords(
		t, "https://unmeasured.example/", 1,
	)
	withoutLinkCounts.facts.AmountOfLinks = yacymodel.None[int]()
	answers := answersOf(
		[]string{"berlin"},
		[]answeredDocument{
			foundDocumentWithLinksAmongAThousandWords(t, "https://sparse.example/", 0),
			withoutLinkCounts,
			foundDocumentWithLinksAmongAThousandWords(t, "https://linked.example/", 50),
		},
	)

	want := []string{
		"https://linked.example/",
		"https://unmeasured.example/",
		"https://sparse.example/",
	}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTheDocumentOfLinksEnoughKeepsTheRelevanceOfAFurtherLinkedDocument(t *testing.T) {
	t.Parallel()

	answers := answersOf(
		[]string{"berlin"},
		[]answeredDocument{
			foundDocumentWithLinksAmongAThousandWords(t, "https://linked.example/", 30),
			foundDocumentWithLinksAmongAThousandWords(t, "https://further.linked.example/", 600),
		},
	)

	relevancePerDocument := documentrelevance.New(documentrelevance.DefaultScoreWeights()).
		RelevancePerDocumentOf(answers)
	linked := relevancePerDocument[answers.FoundDocuments[0].Hash]
	furtherLinked := relevancePerDocument[answers.FoundDocuments[1].Hash]
	if linked != furtherLinked {
		t.Fatalf(
			"the document of links enough reaches the relevance %.4f, want the %.4f of the "+
				"further linked document",
			linked,
			furtherLinked,
		)
	}
}

func foundDocumentNobodyCountedTheQueryPhrasesOf(
	t *testing.T, address string,
) answeredDocument {
	t.Helper()

	return foundDocumentWithHitsOf(
		t, address, "berlin", hitsOfTheQueryWordInEveryPhrasedDocument,
	)
}

func TestTheDocumentNobodyCountedTheQueryPhrasesOfKeepsTheRelevanceOfAnApartDocument(t *testing.T) {
	t.Parallel()

	uncounted := foundDocumentNobodyCountedTheQueryPhrasesOf(t, "https://uncounted.example/")
	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]string{"berlin"},
		[]answeredDocument{
			uncounted,
			foundDocumentHoldingTheQueryPhrase(t, "https://apart.example/", 0),
		},
	)

	relevancePerDocument := documentrelevance.New(documentrelevance.DefaultScoreWeights()).
		RelevancePerDocumentOf(answers)
	apart := relevancePerDocument[answers.FoundDocuments[1].Hash]
	if uncountedRelevance := relevancePerDocument[uncounted.foundDocument.Hash]; uncountedRelevance < apart {
		t.Fatalf(
			"the document nobody counted the query phrases of reaches the relevance %.4f, want "+
				"no less than the %.4f of the document that holds the query words apart",
			uncountedRelevance,
			apart,
		)
	}
}

func TestTheDocumentNobodyCountedTheQueryPhrasesOfComesAfterThePhrasedOne(
	t *testing.T,
) {
	t.Parallel()

	answers := answersHolding(
		map[string]int{"berlin": 100},
		[]string{"berlin"},
		[]answeredDocument{
			foundDocumentNobodyCountedTheQueryPhrasesOf(t, "https://uncounted.example/"),
			foundDocumentHoldingTheQueryPhrase(t, "https://phrased.example/", 9),
		},
	)

	want := []string{"https://phrased.example/", "https://uncounted.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func foundDocumentThisNodeCountedTheWordsOf(
	t *testing.T, address string, amountOfWords int,
) answeredDocument {
	t.Helper()

	foundDocument := foundDocumentWithHitsOf(t, address, "berlin", 3)
	foundDocument.facts.AmountOfWords = yacymodel.Some(amountOfWords)

	return foundDocument
}

func foundDocumentNobodyCountedTheWordsOf(t *testing.T, address string) answeredDocument {
	t.Helper()

	answered := foundDocumentAt(t, address)
	postingReplicasPerDocument := queryanswers.PostingReplicasPerDocument{}
	postingReplicasPerDocument.Keep(answered.foundDocument.Hash, queryanswers.PostingReplica{
		Word: yacymodel.Some(yacymodel.WordHash("berlin")),
		Posting: yacymodel.RWIPosting{
			Hits:      hitsOfTheQueryWordInEveryUnreadDocument,
			TextWords: amountOfWordsAPeerCountedOfAnUnreadDocument,
		},
	})
	answered.facts = queryanswers.FactsPerDocumentOf(
		postingReplicasPerDocument,
	)[answered.foundDocument.Hash]

	return answered
}

func TestTheRelevanceOfAReadDocumentHoldsHoweverManyUnreadDocumentsTheAnswersHold(t *testing.T) {
	t.Parallel()

	readDocuments := []answeredDocument{
		foundDocumentThisNodeCountedTheWordsOf(t, "https://short.example/", 400),
		foundDocumentThisNodeCountedTheWordsOf(t, "https://long.example/", 12000),
	}
	documentsPerWord := map[string]int{"berlin": 100}
	amongTheReadDocuments := documentrelevance.New(documentrelevance.DefaultScoreWeights()).
		RelevancePerDocumentOf(answersHolding(documentsPerWord, []string{"berlin"}, readDocuments))
	amongTheUnreadDocumentsToo := documentrelevance.New(documentrelevance.DefaultScoreWeights()).
		RelevancePerDocumentOf(answersHolding(documentsPerWord, []string{"berlin"}, append(
			slices.Clone(readDocuments),
			foundDocumentNobodyCountedTheWordsOf(t, "https://unread.example/"),
			foundDocumentNobodyCountedTheWordsOf(t, "https://further-unread.example/"),
		)))

	for _, readDocument := range readDocuments {
		beside := amongTheUnreadDocumentsToo[readDocument.foundDocument.Hash]
		without := amongTheReadDocuments[readDocument.foundDocument.Hash]
		if beside == without {
			continue
		}
		t.Fatalf(
			"the document %q reaches the relevance %.4f beside the unread documents, want the "+
				"%.4f it reaches without them",
			readDocument.foundDocument.Address,
			beside,
			without,
		)
	}
}
