package queryanswers_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const addressOfTheReadDocument = "https://berlin.example/"

func documentOf(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return hash
}

func answersOfTheDocumentAPeerMatchedForTheWord(
	t *testing.T, address string, word string,
) queryanswers.AnsweredQuery {
	t.Helper()

	document := documentOf(t, address)

	return queryanswers.AnsweredQuery{
		QueryWords: []yacymodel.Hash{yacymodel.WordHash(word)},
		FoundDocuments: []queryanswers.FoundDocument{{
			Hash:    document,
			Address: address,
			Title:   "The title a peer sent",
		}},
		FactsPerDocument: queryanswers.FactsPerDocument{document: {
			HitsPerQueryWord: map[yacymodel.Hash]int{yacymodel.WordHash(word): 1},
			AmountOfLinks:    yacymodel.Some(2),
		}},
		DocumentsHeldPerQueryWord: map[yacymodel.Hash]int{yacymodel.WordHash(word): 12},
	}
}

func answersOfTheReadDocument(t *testing.T) queryanswers.AnsweredQuery {
	t.Helper()

	return answersOfTheDocumentAPeerMatchedForTheWord(t, addressOfTheReadDocument, "berlin")
}

func pageContentsOfTheReadDocument(
	t *testing.T,
) map[yacymodel.URLHash]pagecontents.PageContents {
	t.Helper()

	return map[yacymodel.URLHash]pagecontents.PageContents{
		documentOf(t, addressOfTheReadDocument): {
			Title:            "Berlin",
			HitsPerQueryWord: map[yacymodel.Hash]int{yacymodel.WordHash("berlin"): 7},
			AmountOfWords:    400,
			Snippet:          "Berlin holds a wall.",
			LinkCounts:       pagecontents.LinkCounts{LocalLinks: 25, ExternalLinks: 4},
		},
	}
}

func TestAReadDocumentCarriesWhatItsTextHolds(t *testing.T) {
	t.Parallel()

	answers := answersOfTheReadDocument(t)

	read := answers.WithTheContentsOfTheReadPages(pageContentsOfTheReadDocument(t))

	facts := read.FactsPerDocument[documentOf(t, addressOfTheReadDocument)]
	foundDocument := read.FoundDocuments[0]
	if facts.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 7 ||
		facts.AmountOfWords.OrElse(0) != 400 ||
		foundDocument.Snippet != "Berlin holds a wall." ||
		foundDocument.Title != "Berlin" {
		t.Fatalf(
			"the document reads %+v and %+v, want the counts, the snippet and the title of the "+
				"text",
			foundDocument,
			facts,
		)
	}
}

func TestTheTextOfADocumentCountsAQueryWordNoPeerMatchedItFor(t *testing.T) {
	t.Parallel()

	answers := answersOfTheReadDocument(t)
	answers.QueryWords = append(answers.QueryWords, yacymodel.WordHash("weather"))
	pageContentsOfTheDocument := pageContentsOfTheReadDocument(t)
	pageContents := pageContentsOfTheDocument[documentOf(t, addressOfTheReadDocument)]
	pageContents.HitsPerQueryWord[yacymodel.WordHash("weather")] = 2
	pageContentsOfTheDocument[documentOf(t, addressOfTheReadDocument)] = pageContents

	read := answers.WithTheContentsOfTheReadPages(pageContentsOfTheDocument)

	facts := read.FactsPerDocument[documentOf(t, addressOfTheReadDocument)]
	if facts.HitsPerQueryWord[yacymodel.WordHash("weather")] != 2 ||
		facts.AmountOfWords.OrElse(0) != 400 {
		t.Fatalf(
			"the document reads %+v for the word no peer matched it for, want the count of the "+
				"text",
			facts,
		)
	}
}

func TestAReadPageWithoutATitleKeepsTheTitleAPeerSent(t *testing.T) {
	t.Parallel()

	answers := answersOfTheReadDocument(t)
	pageContentsWithoutATitle := pageContentsOfTheReadDocument(t)
	pageContents := pageContentsWithoutATitle[documentOf(t, addressOfTheReadDocument)]
	pageContents.Title = ""
	pageContentsWithoutATitle[documentOf(t, addressOfTheReadDocument)] = pageContents

	read := answers.WithTheContentsOfTheReadPages(pageContentsWithoutATitle)

	if read.FoundDocuments[0].Title != "The title a peer sent" {
		t.Fatalf(
			"the found document holds the title %q, want the title the peer sent",
			read.FoundDocuments[0].Title,
		)
	}
}

func TestTheDocumentsHeldPerQueryWordStayAsThePeersCountedThem(t *testing.T) {
	t.Parallel()

	answers := answersOfTheReadDocument(t)

	read := answers.WithTheContentsOfTheReadPages(pageContentsOfTheReadDocument(t))

	if read.DocumentsHeldPerQueryWord[yacymodel.WordHash("berlin")] != 12 {
		t.Fatalf(
			"the answers hold %+v, want the documents the peers counted",
			read.DocumentsHeldPerQueryWord,
		)
	}
}

func TestADocumentThatWasNotReadStaysAsThePeersCountedIt(t *testing.T) {
	t.Parallel()

	answers := answersOfTheDocumentAPeerMatchedForTheWord(
		t, "https://unread.example/", "berlin",
	)

	read := answers.WithTheContentsOfTheReadPages(pageContentsOfTheReadDocument(t))

	facts := read.FactsPerDocument[documentOf(t, "https://unread.example/")]
	if facts.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 1 ||
		facts.AmountOfWords.Present() ||
		facts.AmountOfLinks.OrElse(0) != 2 {
		t.Fatalf("the document reads %+v, want the counts the peers answered", facts)
	}
}

func TestAReadPageThatMovedGivesItsDocumentTheAddressItMovedTo(t *testing.T) {
	t.Parallel()

	answers := answersOfTheReadDocument(t)
	pageContentsOfAMovedPage := pageContentsOfTheReadDocument(t)
	pageContents := pageContentsOfAMovedPage[documentOf(t, addressOfTheReadDocument)]
	pageContents.Address = "https://berlin.example/moved"
	pageContentsOfAMovedPage[documentOf(t, addressOfTheReadDocument)] = pageContents

	read := answers.WithTheContentsOfTheReadPages(pageContentsOfAMovedPage)

	foundDocument := read.FoundDocuments[0]
	if foundDocument.Address != "https://berlin.example/moved" ||
		foundDocument.Hash != documentOf(t, addressOfTheReadDocument) {
		t.Fatalf(
			"the found document reads %+v, want the address the page moved to under its own hash",
			foundDocument,
		)
	}
}

func TestAReadPageThatDidNotMoveKeepsTheAddressAPeerSent(t *testing.T) {
	t.Parallel()

	read := answersOfTheReadDocument(
		t,
	).WithTheContentsOfTheReadPages(pageContentsOfTheReadDocument(t))

	if read.FoundDocuments[0].Address != addressOfTheReadDocument {
		t.Fatalf(
			"the found document holds the address %q, want the address the peer sent",
			read.FoundDocuments[0].Address,
		)
	}
}

func TestAReadPageGivesItsDocumentTheLinksItHoldsInPlaceOfTheLinksAPeerCounted(t *testing.T) {
	t.Parallel()

	answers := answersOfTheReadDocument(t)

	read := answers.WithTheContentsOfTheReadPages(pageContentsOfTheReadDocument(t))

	facts := read.FactsPerDocument[documentOf(t, addressOfTheReadDocument)]
	if amountOfLinks, counted := facts.AmountOfLinks.Get(); !counted || amountOfLinks != 29 {
		t.Fatalf(
			"the document holds %d links counted %t, want the 29 links the page holds",
			amountOfLinks,
			counted,
		)
	}
}

func TestAReadPageCountsAQueryWordItHoldsNoneOfInPlaceOfThePeerThatMatchedIt(t *testing.T) {
	t.Parallel()

	answers := answersOfTheDocumentAPeerMatchedForTheWord(
		t, addressOfTheReadDocument, "weather",
	)

	read := answers.WithTheContentsOfTheReadPages(pageContentsOfTheReadDocument(t))

	facts := read.FactsPerDocument[documentOf(t, addressOfTheReadDocument)]
	if facts.HitsPerQueryWord[yacymodel.WordHash("weather")] != 0 {
		t.Fatalf(
			"the document counts %d hits of the query word its read page holds none of, want the "+
				"count of the text",
			facts.HitsPerQueryWord[yacymodel.WordHash("weather")],
		)
	}
}

func TestOnlyAReadPageCountsTheQueryPhrases(t *testing.T) {
	t.Parallel()

	answers := answersOfTheReadDocument(t)
	unread := answersOfTheDocumentAPeerMatchedForTheWord(t, "https://unread.example/", "berlin")
	answers.FoundDocuments = append(answers.FoundDocuments, unread.FoundDocuments[0])
	for document, facts := range unread.FactsPerDocument {
		answers.FactsPerDocument[document] = facts
	}
	pageContentsOfTheDocument := pageContentsOfTheReadDocument(t)
	pageContents := pageContentsOfTheDocument[documentOf(t, addressOfTheReadDocument)]
	pageContents.QueryPhraseHits = 3
	pageContentsOfTheDocument[documentOf(t, addressOfTheReadDocument)] = pageContents

	read := answers.WithTheContentsOfTheReadPages(pageContentsOfTheDocument)

	queryPhraseHitsOfTheReadDocument := read.
		FactsPerDocument[documentOf(t, addressOfTheReadDocument)].QueryPhraseHits
	queryPhraseHitsOfTheUnreadDocument := read.
		FactsPerDocument[documentOf(t, "https://unread.example/")].QueryPhraseHits
	if queryPhraseHitsOfTheReadDocument.OrElse(0) != 3 ||
		queryPhraseHitsOfTheUnreadDocument.Present() {
		t.Fatalf(
			"the read document holds %+v query phrase hits and the unread one %+v, want the "+
				"three hits of the read page and none for the unread document",
			queryPhraseHitsOfTheReadDocument,
			queryPhraseHitsOfTheUnreadDocument,
		)
	}
}
