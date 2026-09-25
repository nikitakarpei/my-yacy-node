package documentrelevance_test

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	amountOfRunsOfSameAnswers     = 50
	amountOfWordsOfLinkedDocument = 1000

	hitsOfQueryWordInEveryUnreadDocument  = 3
	hitsOfQueryWordInEveryPhrasedDocument = 3

	malformedAddress = "https://berlin weather.example/"
)

type foundDocument queryanswers.FoundDocument

func foundDocumentAt(t *testing.T, address string) foundDocument {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return foundDocument{
		Hash:    hash,
		Address: address,
		Facts:   queryanswers.DocumentFacts{HitsPerQueryWord: map[yacymodel.Hash]int{}},
	}
}

func (document foundDocument) matchingWords(words ...string) foundDocument {
	for _, word := range words {
		document.Facts.HitsPerQueryWord[yacymodel.WordHash(word)] = 0
	}

	return document
}

func (document foundDocument) withHitsOf(word string, hits int) foundDocument {
	document.Facts.HitsPerQueryWord[yacymodel.WordHash(word)] = hits

	return document
}

func (document foundDocument) withTitle(title string) foundDocument {
	document.Title = title

	return document
}

func (document foundDocument) withAmountOfWords(amountOfWords int) foundDocument {
	document.Facts.AmountOfWords = yacymodel.Some(amountOfWords)

	return document
}

func (document foundDocument) withAmountOfLinks(amountOfLinks int) foundDocument {
	document.Facts.AmountOfLinks = yacymodel.Some(amountOfLinks)

	return document
}

func (document foundDocument) withQueryPhraseHits(queryPhraseHits int) foundDocument {
	document.Facts.QueryPhraseHits = yacymodel.Some(queryPhraseHits)

	return document
}

func (document foundDocument) shownAtAddress(address string) foundDocument {
	document.Address = address

	return document
}

func (document foundDocument) withHitsAPeerCounted(word string, hits int) foundDocument {
	document.PostingReplicas = append(
		slices.Clone(document.PostingReplicas),
		queryanswers.PostingReplica{
			Word:    yacymodel.WordHash(word),
			Posting: yacymodel.RWIPosting{Hits: hits},
		},
	)
	document.Facts = queryanswers.FoundDocumentOf(
		document.Hash,
		nil,
		document.PostingReplicas,
	).Facts

	return document
}

func answersOf(
	queryWords []string,
	foundDocuments []foundDocument,
) queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords:     hashesOf(queryWords),
		FoundDocuments: foundDocumentsOf(foundDocuments),
	}
}

func hashesOf(queryWords []string) []yacymodel.Hash {
	hashes := make([]yacymodel.Hash, 0, len(queryWords))
	for _, queryWord := range queryWords {
		hashes = append(hashes, yacymodel.WordHash(queryWord))
	}

	return hashes
}

func foundDocumentsOf(foundDocuments []foundDocument) []queryanswers.FoundDocument {
	documents := make([]queryanswers.FoundDocument, 0, len(foundDocuments))
	for _, document := range foundDocuments {
		documents = append(documents, queryanswers.FoundDocument(document))
	}

	return documents
}

func answersHoldingDocumentsPerQueryWord(
	queryWords []string,
	foundDocuments []foundDocument,
	documentsHeldPerWord map[string]int,
) queryanswers.AnsweredQuery {
	documentsHeldPerQueryWord := make(map[yacymodel.Hash]int, len(documentsHeldPerWord))
	for word, documentsHeldForWord := range documentsHeldPerWord {
		documentsHeldPerQueryWord[yacymodel.WordHash(word)] = documentsHeldForWord
	}

	return queryanswers.AnsweredQuery{
		QueryWords:                hashesOf(queryWords),
		FoundDocuments:            foundDocumentsOf(foundDocuments),
		DocumentsHeldPerQueryWord: documentsHeldPerQueryWord,
	}
}

func addressesInFallingOrderOfRelevance(answers queryanswers.AnsweredQuery) []string {
	return addressesInFallingOrderOfRelevanceBy(
		documentrelevance.DefaultRelevanceWeights(), answers,
	)
}

func addressesInFallingOrderOfRelevanceBy(
	relevanceWeights documentrelevance.RelevanceWeights, answers queryanswers.AnsweredQuery,
) []string {
	relevancePerDocument := documentrelevance.RelevanceScorerWeighedBy(relevanceWeights).
		RelevancePerDocumentOf(context.Background(), answers)
	foundDocuments := slices.Clone(answers.FoundDocuments)
	slices.SortStableFunc(foundDocuments, func(one, other queryanswers.FoundDocument) int {
		return cmp.Compare(
			relevancePerDocument[other.Hash], relevancePerDocument[one.Hash],
		)
	})

	addresses := make([]string, 0, len(foundDocuments))
	for _, document := range foundDocuments {
		addresses = append(addresses, document.Address)
	}

	return addresses
}

func TestDocumentOfRarerQueryWordComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"berlin", "kelondro"},
		[]foundDocument{
			foundDocumentAt(t, "https://common.example/").withHitsOf("berlin", 1),
			foundDocumentAt(t, "https://rare.example/").withHitsOf("kelondro", 1),
		},
		map[string]int{"berlin": 100000, "kelondro": 10},
	)

	want := []string{"https://rare.example/", "https://common.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestEveryQueryWordWeighsSameWhenNoPeerCountedDocumentsForIt(t *testing.T) {
	t.Parallel()

	answers := answersOf(
		[]string{"berlin", "kelondro"},
		[]foundDocument{
			foundDocumentAt(t, "https://a.example/").withHitsOf("berlin", 1),
			foundDocumentAt(t, "https://b.example/").withHitsOf("kelondro", 1),
		},
	)

	want := []string{"https://a.example/", "https://b.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want the found order %v", got, want)
	}
}

func TestWordNoPeerCountedDocumentsForWeighsAsMuchAsMostCommonCountedWord(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"berlin", "kelondro"},
		[]foundDocument{
			foundDocumentAt(t, "https://counted.example/").withHitsOf("berlin", 1),
			foundDocumentAt(t, "https://uncounted.example/").withHitsOf("kelondro", 1),
		},
		map[string]int{"berlin": 100},
	)

	want := []string{"https://counted.example/", "https://uncounted.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want the found order %v", got, want)
	}
}

func TestDocumentWithMoreHitsOfSameWordComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"berlin"},
		[]foundDocument{
			foundDocumentAt(t, "https://once.example/").withHitsOf("berlin", 1),
			foundDocumentAt(t, "https://often.example/").withHitsOf("berlin", 9),
		},
		map[string]int{"berlin": 100},
	)

	want := []string{"https://often.example/", "https://once.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestEachFurtherHitOfSameWordAddsLessThanFirstHitOfAnotherWord(t *testing.T) {
	t.Parallel()

	twiceOneWord := foundDocumentAt(t, "https://one-word.example/").withHitsOf("berlin", 2)
	onceEachWord := foundDocumentAt(t, "https://two-words.example/").
		withHitsOf("berlin", 1).
		withHitsOf("weather", 1)
	answers := answersOf(
		[]string{"berlin", "weather"},
		[]foundDocument{
			twiceOneWord,
			onceEachWord,
		},
	)

	want := []string{"https://two-words.example/", "https://one-word.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestShorterDocumentOfSameHitsComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"berlin"},
		[]foundDocument{
			foundDocumentAt(t, "https://long.example/").
				withHitsOf("berlin", 3).
				withAmountOfWords(5000),
			foundDocumentAt(t, "https://short.example/").
				withHitsOf("berlin", 3).
				withAmountOfWords(100),
		},
		map[string]int{"berlin": 100},
	)

	want := []string{"https://short.example/", "https://long.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func relevanceOfDocumentAt(
	t *testing.T, answers queryanswers.AnsweredQuery, address string,
) float64 {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return documentrelevance.RelevanceScorerWeighedBy(documentrelevance.DefaultRelevanceWeights()).
		RelevancePerDocumentOf(t.Context(), answers)[hash]
}

func TestDocumentOfHitOfEveryQueryWordHoldsSameRelevanceHoweverLongTheQueryIs(t *testing.T) {
	t.Parallel()

	const address = "https://a.example/"
	relevanceUnderShortQuery := relevanceOfDocumentAt(t, answersHoldingDocumentsPerQueryWord(
		[]string{"berlin"},
		[]foundDocument{
			foundDocumentAt(t, address).withHitsOf("berlin", 1),
		},
		map[string]int{"berlin": 100},
	), address)
	relevanceUnderLongQuery := relevanceOfDocumentAt(t, answersHoldingDocumentsPerQueryWord(
		[]string{"berlin", "weather", "today"},
		[]foundDocument{
			foundDocumentAt(t, address).
				withHitsOf("berlin", 1).
				withHitsOf("weather", 1).
				withHitsOf("today", 1),
		},
		map[string]int{"berlin": 100, "weather": 100, "today": 100},
	), address)

	if relevanceUnderShortQuery != relevanceUnderLongQuery {
		t.Fatalf(
			"the document holds the relevance %f under the query of one word and %f under the "+
				"query of three words, want the same relevance",
			relevanceUnderShortQuery, relevanceUnderLongQuery,
		)
	}
}

func TestDocumentThatMatchedMoreQueryWordsComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"berlin", "weather"},
		[]foundDocument{
			foundDocumentAt(t, "https://one.example/").withHitsOf("berlin", 1),
			foundDocumentAt(t, "https://both.example/").
				withHitsOf("berlin", 1).
				withHitsOf("weather", 1),
		},
		map[string]int{"berlin": 100, "weather": 100},
	)

	want := []string{"https://both.example/", "https://one.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestDocumentWhoseHostHoldsTheOnlyQueryWordComesBeforeOneOfHitsOfThatWord(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"berlin"},
		[]foundDocument{
			foundDocumentAt(t, "https://berlin.example/").withHitsOf("berlin", 0),
			foundDocumentAt(t, "https://weather.example/").withHitsOf("berlin", 5),
		},
		map[string]int{"berlin": 100},
	)

	want := []string{"https://berlin.example/", "https://weather.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestUnreadDocumentWithoutHitComesAfterOneWithHit(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"berlin", "weather"},
		[]foundDocument{
			foundDocumentAt(t, "https://uncounted.example/").
				matchingWords("berlin", "weather"),
			foundDocumentAt(t, "https://counted.example/").withHitsOf("berlin", 1),
		},
		map[string]int{"berlin": 100, "weather": 100},
	)

	want := []string{"https://counted.example/", "https://uncounted.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestDocumentWithoutTitleComesAfterOneAPeerPutBehindIt(t *testing.T) {
	t.Parallel()

	answers := answersOf(
		[]string{"berlin"},
		[]foundDocument{
			foundDocumentAt(t, "https://untitled.example/").matchingWords("berlin"),
			foundDocumentAt(t, "https://titled.example/").
				matchingWords("berlin").
				withTitle("A city"),
		},
	)

	want := []string{"https://titled.example/", "https://untitled.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestDocumentWhoseTitleHoldsQueryWordComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"berlin"},
		[]foundDocument{
			foundDocumentAt(t, "https://beside.example/").
				matchingWords("berlin").
				withTitle("The weather of a city"),
			foundDocumentAt(t, "https://titled.example/").
				matchingWords("berlin").
				withTitle("Berlin weather"),
		},
		map[string]int{"berlin": 100},
	)

	want := []string{"https://titled.example/", "https://beside.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestDocumentWhoseTitleHoldsRarerQueryWordComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"emacs", "manual"},
		[]foundDocument{
			foundDocumentAt(t, "https://beside.example/").
				matchingWords("emacs", "manual").
				withTitle("The manual"),
			foundDocumentAt(t, "https://titled.example/").
				matchingWords("emacs", "manual").
				withTitle("The emacs"),
		},
		map[string]int{"emacs": 10, "manual": 100000},
	)

	want := []string{"https://titled.example/", "https://beside.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTitleThatOnlyHoldsLongerWordChangesNoOrder(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"tofu"},
		[]foundDocument{
			foundDocumentAt(t, "https://beside.example/").
				matchingWords("tofu").
				withTitle("Migrating to a fork"),
			foundDocumentAt(t, "https://titled.example/").
				matchingWords("tofu").
				withTitle("Migrating to OpenTofu"),
		},
		map[string]int{"tofu": 100},
	)

	want := []string{"https://beside.example/", "https://titled.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTitleIsReadPastItsPunctuation(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"opentofu"},
		[]foundDocument{
			foundDocumentAt(t, "https://beside.example/").matchingWords("opentofu"),
			foundDocumentAt(t, "https://titled.example/").
				matchingWords("opentofu").
				withTitle("Terraform vs. OpenTofu: what changed?"),
		},
		map[string]int{"opentofu": 100},
	)

	want := []string{"https://titled.example/", "https://beside.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestDocumentWhoseTextHoldsQueryPhraseComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"berlin"},
		[]foundDocument{
			foundDocumentAt(t, "https://apart.example/").
				withHitsOf("berlin", hitsOfQueryWordInEveryPhrasedDocument).
				withQueryPhraseHits(0),
			foundDocumentAt(t, "https://phrased.example/").
				withHitsOf("berlin", hitsOfQueryWordInEveryPhrasedDocument).
				withQueryPhraseHits(1),
		},
		map[string]int{"berlin": 100},
	)

	want := []string{"https://phrased.example/", "https://apart.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestEachFurtherQueryPhraseHitAddsLessThanFirst(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"berlin"},
		[]foundDocument{
			foundDocumentAt(t, "https://once.example/").
				withHitsOf("berlin", hitsOfQueryWordInEveryPhrasedDocument).
				withQueryPhraseHits(1),
			foundDocumentAt(t, "https://often.example/").
				withHitsOf("berlin", hitsOfQueryWordInEveryPhrasedDocument).
				withQueryPhraseHits(9),
		},
		map[string]int{"berlin": 100},
	)

	want := []string{"https://often.example/", "https://once.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestDocumentOfMalformedAddressKeepsPlaceItWasFoundAt(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"berlin"},
		[]foundDocument{
			foundDocumentAt(t, "https://unreadable.example/").
				matchingWords("berlin").
				shownAtAddress(malformedAddress),
			foundDocumentAt(t, "https://weather.example/city/").matchingWords("berlin"),
		},
		map[string]int{"berlin": 100},
	)

	want := []string{malformedAddress, "https://weather.example/city/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want the found order %v", got, want)
	}
}

func TestSameAnswersComeBackInSameOrderEveryRun(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"berlin", "weather", "kelondro", "freeworld"},
		[]foundDocument{
			foundDocumentAt(t, "https://a.example/").
				withHitsOf("berlin", 3).withHitsOf("weather", 2).
				withHitsOf("kelondro", 1).withHitsOf("freeworld", 4),
			foundDocumentAt(t, "https://b.example/").
				withHitsOf("berlin", 4).withHitsOf("weather", 1).
				withHitsOf("kelondro", 1).withHitsOf("freeworld", 3),
			foundDocumentAt(t, "https://c.example/").
				withHitsOf("berlin", 2).withHitsOf("weather", 3).
				withHitsOf("kelondro", 2).withHitsOf("freeworld", 1),
			foundDocumentAt(t, "https://d.example/").
				withHitsOf("berlin", 1).withHitsOf("weather", 4).
				withHitsOf("kelondro", 1).withHitsOf("freeworld", 2),
			foundDocumentAt(t, "https://e.example/").
				withHitsOf("berlin", 3).withHitsOf("weather", 3).
				withHitsOf("kelondro", 1).withHitsOf("freeworld", 1),
		},
		map[string]int{"berlin": 100000, "weather": 7000, "kelondro": 13, "freeworld": 421},
	)

	want := addressesInFallingOrderOfRelevance(answers)
	for run := range amountOfRunsOfSameAnswers {
		if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
			t.Fatalf("run %d reads the relevance order %v, want %v", run, got, want)
		}
	}
}

func TestTwoDocumentsOfSameCountedHitsHoldSameRelevance(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"berlin"},
		[]foundDocument{
			foundDocumentAt(t, "https://a.example/").withHitsOf("berlin", 1),
			foundDocumentAt(t, "https://b.example/").withHitsOf("berlin", 1),
		},
		map[string]int{"berlin": 100},
	)

	relevanceScorer := documentrelevance.RelevanceScorerWeighedBy(
		documentrelevance.DefaultRelevanceWeights(),
	)
	relevancePerDocument := relevanceScorer.RelevancePerDocumentOf(t.Context(), answers)
	relevanceOfDocuments := slices.Collect(maps.Values(relevancePerDocument))
	if len(relevanceOfDocuments) != 2 ||
		relevanceOfDocuments[0] != relevanceOfDocuments[1] {
		t.Fatalf(
			"the relevance per document reads %v, want the same relevance for both documents",
			relevancePerDocument,
		)
	}
}

func TestEntryPageOfSiteTheQueryHoldsComesFirst(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"heise"},
		[]foundDocument{
			foundDocumentAt(t, "https://www.heise.de/developer/kontakt/").matchingWords("heise"),
			foundDocumentAt(t, "https://heise-academy.de/").matchingWords("heise"),
			foundDocumentAt(t, "https://www.heise.de/").matchingWords("heise"),
		},
		map[string]int{"heise": 100},
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

func TestDocumentOfFewerLinksPerWordComesLast(t *testing.T) {
	t.Parallel()

	answers := answersOf(
		[]string{"berlin"},
		[]foundDocument{
			foundDocumentAt(t, "https://sparse.example/").
				withHitsOf("berlin", 1).
				withAmountOfWords(amountOfWordsOfLinkedDocument).
				withAmountOfLinks(1),
			foundDocumentAt(t, "https://linked.example/").
				withHitsOf("berlin", 1).
				withAmountOfWords(amountOfWordsOfLinkedDocument).
				withAmountOfLinks(50),
		},
	)

	want := []string{"https://linked.example/", "https://sparse.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestDocumentNoPeerSentLinkCountsForComesBetweenTheSparseAndTheLinked(t *testing.T) {
	t.Parallel()

	answers := answersOf(
		[]string{"berlin"},
		[]foundDocument{
			foundDocumentAt(t, "https://sparse.example/").
				withHitsOf("berlin", 1).
				withAmountOfWords(amountOfWordsOfLinkedDocument).
				withAmountOfLinks(0),
			foundDocumentAt(t, "https://unmeasured.example/").
				withHitsOf("berlin", 1).
				withAmountOfWords(amountOfWordsOfLinkedDocument),
			foundDocumentAt(t, "https://linked.example/").
				withHitsOf("berlin", 1).
				withAmountOfWords(amountOfWordsOfLinkedDocument).
				withAmountOfLinks(50),
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

func TestDocumentOfLinksEnoughKeepsRelevanceOfFurtherLinkedDocument(t *testing.T) {
	t.Parallel()

	answers := answersOf(
		[]string{"berlin"},
		[]foundDocument{
			foundDocumentAt(t, "https://linked.example/").
				withHitsOf("berlin", 1).
				withAmountOfWords(amountOfWordsOfLinkedDocument).
				withAmountOfLinks(30),
			foundDocumentAt(t, "https://further.linked.example/").
				withHitsOf("berlin", 1).
				withAmountOfWords(amountOfWordsOfLinkedDocument).
				withAmountOfLinks(600),
		},
	)

	relevanceScorer := documentrelevance.RelevanceScorerWeighedBy(
		documentrelevance.DefaultRelevanceWeights(),
	)
	relevancePerDocument := relevanceScorer.RelevancePerDocumentOf(t.Context(), answers)
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

func TestUnreadDocumentKeepsRelevanceOfApartDocument(t *testing.T) {
	t.Parallel()

	unread := foundDocumentAt(t, "https://uncounted.example/").
		withHitsOf("berlin", hitsOfQueryWordInEveryPhrasedDocument)
	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"berlin"},
		[]foundDocument{
			unread,
			foundDocumentAt(t, "https://apart.example/").
				withHitsOf("berlin", hitsOfQueryWordInEveryPhrasedDocument).
				withQueryPhraseHits(0),
		},
		map[string]int{"berlin": 100},
	)

	relevanceScorer := documentrelevance.RelevanceScorerWeighedBy(
		documentrelevance.DefaultRelevanceWeights(),
	)
	relevancePerDocument := relevanceScorer.RelevancePerDocumentOf(t.Context(), answers)
	apart := relevancePerDocument[answers.FoundDocuments[1].Hash]
	if unreadRelevance := relevancePerDocument[unread.Hash]; unreadRelevance < apart {
		t.Fatalf(
			"the unread document reaches the relevance %.4f, want no less than the %.4f of the "+
				"document that holds the query words apart",
			unreadRelevance,
			apart,
		)
	}
}

func TestUnreadDocumentComesAfterPhrasedOne(t *testing.T) {
	t.Parallel()

	answers := answersHoldingDocumentsPerQueryWord(
		[]string{"berlin"},
		[]foundDocument{
			foundDocumentAt(t, "https://uncounted.example/").
				withHitsOf("berlin", hitsOfQueryWordInEveryPhrasedDocument),
			foundDocumentAt(t, "https://phrased.example/").
				withHitsOf("berlin", hitsOfQueryWordInEveryPhrasedDocument).
				withQueryPhraseHits(9),
		},
		map[string]int{"berlin": 100},
	)

	want := []string{"https://phrased.example/", "https://uncounted.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestRelevanceOfReadDocumentHoldsHoweverManyUnreadDocumentsTheAnswersHold(t *testing.T) {
	t.Parallel()

	readDocuments := []foundDocument{
		foundDocumentAt(t, "https://short.example/").
			withHitsOf("berlin", 3).
			withAmountOfWords(400),
		foundDocumentAt(t, "https://long.example/").
			withHitsOf("berlin", 3).
			withAmountOfWords(12000),
	}
	documentsHeldPerWord := map[string]int{"berlin": 100}
	relevanceScorer := documentrelevance.RelevanceScorerWeighedBy(
		documentrelevance.DefaultRelevanceWeights(),
	)
	amongReadDocuments := relevanceScorer.
		RelevancePerDocumentOf(t.Context(), answersHoldingDocumentsPerQueryWord(
			[]string{"berlin"}, readDocuments, documentsHeldPerWord,
		))
	amongUnreadDocumentsToo := relevanceScorer.
		RelevancePerDocumentOf(t.Context(), answersHoldingDocumentsPerQueryWord(
			[]string{"berlin"},
			append(
				slices.Clone(readDocuments),
				foundDocumentAt(t, "https://unread.example/").withHitsAPeerCounted(
					"berlin", hitsOfQueryWordInEveryUnreadDocument,
				),
				foundDocumentAt(t, "https://further-unread.example/").withHitsAPeerCounted(
					"berlin", hitsOfQueryWordInEveryUnreadDocument,
				),
			),
			documentsHeldPerWord,
		))

	for _, readDocument := range readDocuments {
		beside := amongUnreadDocumentsToo[readDocument.Hash]
		without := amongReadDocuments[readDocument.Hash]
		if beside == without {
			continue
		}
		t.Fatalf(
			"the document %q reaches the relevance %.4f beside the unread documents, want the "+
				"%.4f it reaches without them",
			readDocument.Address,
			beside,
			without,
		)
	}
}
