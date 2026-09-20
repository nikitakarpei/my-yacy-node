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

func foundDocumentWithOneHitOf(
	t *testing.T, address string, word string,
) queryanswers.FoundDocument {
	t.Helper()

	return queryanswers.FoundDocument{
		Hash:             documentOf(t, address),
		Address:          address,
		Title:            "The title a peer sent",
		HitsPerQueryWord: map[yacymodel.Hash]int{yacymodel.WordHash(word): 1},
	}
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

func answersOfTheReadDocument(t *testing.T) queryanswers.AnsweredQuery {
	t.Helper()

	return queryanswers.AnsweredQuery{
		QueryWords: []yacymodel.Hash{yacymodel.WordHash("berlin")},
		FoundDocuments: []queryanswers.FoundDocument{
			foundDocumentWithOneHitOf(t, addressOfTheReadDocument, "berlin"),
		},
		DocumentsHeldPerQueryWord: map[yacymodel.Hash]int{yacymodel.WordHash("berlin"): 12},
	}
}

func TestAReadDocumentCarriesWhatItsTextHolds(t *testing.T) {
	t.Parallel()

	answers := answersOfTheReadDocument(t)

	read := answers.WithTheContentsOfTheReadPages(pageContentsOfTheReadDocument(t))

	foundDocument := read.FoundDocuments[0]
	if foundDocument.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 7 ||
		foundDocument.AmountOfWordsOfTheReadPage.OrElse(0) != 400 ||
		foundDocument.Snippet != "Berlin holds a wall." ||
		foundDocument.Title != "Berlin" {
		t.Fatalf(
			"the found document reads %+v, want the counts, the snippet and the title of the text",
			foundDocument,
		)
	}
}

func TestTheTextOfADocumentCountsAQueryWordNoPeerMatchedItFor(t *testing.T) {
	t.Parallel()

	answers := queryanswers.AnsweredQuery{
		QueryWords: []yacymodel.Hash{
			yacymodel.WordHash("berlin"), yacymodel.WordHash("weather"),
		},
		FoundDocuments: []queryanswers.FoundDocument{
			foundDocumentWithOneHitOf(t, addressOfTheReadDocument, "berlin"),
		},
	}
	pageContentsOfTheDocument := map[yacymodel.URLHash]pagecontents.PageContents{
		documentOf(t, addressOfTheReadDocument): {
			HitsPerQueryWord: map[yacymodel.Hash]int{
				yacymodel.WordHash("berlin"):  7,
				yacymodel.WordHash("weather"): 2,
			},
			AmountOfWords: 400,
		},
	}

	read := answers.WithTheContentsOfTheReadPages(pageContentsOfTheDocument)

	foundDocument := read.FoundDocuments[0]
	if foundDocument.HitsPerQueryWord[yacymodel.WordHash("weather")] != 2 ||
		foundDocument.AmountOfWordsOfTheReadPage.OrElse(0) != 400 {
		t.Fatalf(
			"the found document reads %+v for the word no peer matched it for, want the count "+
				"of the text",
			foundDocument,
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

	answers := queryanswers.AnsweredQuery{
		FoundDocuments: []queryanswers.FoundDocument{
			foundDocumentWithOneHitOf(t, "https://unread.example/", "berlin"),
		},
	}

	read := answers.WithTheContentsOfTheReadPages(pageContentsOfTheReadDocument(t))

	foundDocument := read.FoundDocuments[0]
	if foundDocument.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 1 ||
		foundDocument.AmountOfWordsOfTheReadPage.Present() {
		t.Fatalf("the found document reads %+v, want the count the peers answered", foundDocument)
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
	answers.FoundDocuments[0].LinkCounts = yacymodel.Some(pagecontents.LinkCounts{
		LocalLinks:    1,
		ExternalLinks: 1,
	})

	read := answers.WithTheContentsOfTheReadPages(pageContentsOfTheReadDocument(t))

	linkCounts, counted := read.FoundDocuments[0].LinkCounts.Get()
	if !counted || linkCounts.LocalLinks != 25 || linkCounts.ExternalLinks != 4 {
		t.Fatalf(
			"the found document holds the link counts %+v counted %t, want the 25 local and 4 "+
				"external links the page holds",
			linkCounts,
			counted,
		)
	}
}

func TestADocumentThatWasNotReadKeepsTheLinksAPeerCounted(t *testing.T) {
	t.Parallel()

	answers := queryanswers.AnsweredQuery{
		FoundDocuments: []queryanswers.FoundDocument{
			foundDocumentWithOneHitOf(t, "https://unread.example/", "berlin"),
		},
	}
	answers.FoundDocuments[0].LinkCounts = yacymodel.Some(pagecontents.LinkCounts{
		LocalLinks:    1,
		ExternalLinks: 1,
	})

	read := answers.WithTheContentsOfTheReadPages(pageContentsOfTheReadDocument(t))

	linkCounts, counted := read.FoundDocuments[0].LinkCounts.Get()
	if !counted || linkCounts.LocalLinks != 1 || linkCounts.ExternalLinks != 1 {
		t.Fatalf(
			"the found document holds the link counts %+v counted %t, want the 1 local and 1 "+
				"external link the peer counted",
			linkCounts,
			counted,
		)
	}
}

func TestAReadPageCountsAQueryWordItHoldsNoneOfInPlaceOfThePeerThatMatchedIt(t *testing.T) {
	t.Parallel()

	answers := queryanswers.AnsweredQuery{
		QueryWords: []yacymodel.Hash{
			yacymodel.WordHash("berlin"), yacymodel.WordHash("weather"),
		},
		FoundDocuments: []queryanswers.FoundDocument{
			foundDocumentWithOneHitOf(t, addressOfTheReadDocument, "weather"),
		},
	}
	pageContentsWithoutTheQueryWord := map[yacymodel.URLHash]pagecontents.PageContents{
		documentOf(t, addressOfTheReadDocument): {
			HitsPerQueryWord: map[yacymodel.Hash]int{yacymodel.WordHash("berlin"): 7},
			AmountOfWords:    400,
		},
	}

	read := answers.WithTheContentsOfTheReadPages(pageContentsWithoutTheQueryWord)

	if read.FoundDocuments[0].HitsPerQueryWord[yacymodel.WordHash("weather")] != 0 {
		t.Fatalf(
			"the found document counts %d hits of the query word its read page holds none of, "+
				"want the count of the text",
			read.FoundDocuments[0].HitsPerQueryWord[yacymodel.WordHash("weather")],
		)
	}
}

func TestOnlyAReadPageCountsTheQueryPhrases(t *testing.T) {
	t.Parallel()

	answers := answersOfTheReadDocument(t)
	answers.FoundDocuments = append(
		answers.FoundDocuments, foundDocumentWithOneHitOf(t, "https://unread.example/", "berlin"),
	)
	pageContentsOfTheDocument := pageContentsOfTheReadDocument(t)
	pageContents := pageContentsOfTheDocument[documentOf(t, addressOfTheReadDocument)]
	pageContents.QueryPhraseHits = 3
	pageContentsOfTheDocument[documentOf(t, addressOfTheReadDocument)] = pageContents

	read := answers.WithTheContentsOfTheReadPages(pageContentsOfTheDocument)

	if read.FoundDocuments[0].QueryPhraseHitsOfTheReadPage.OrElse(0) != 3 ||
		read.FoundDocuments[1].QueryPhraseHitsOfTheReadPage.Present() {
		t.Fatalf(
			"the read document holds %+v query phrase hits and the unread one %+v, want the "+
				"three hits of the read page and none for the unread document",
			read.FoundDocuments[0].QueryPhraseHitsOfTheReadPage,
			read.FoundDocuments[1].QueryPhraseHitsOfTheReadPage,
		)
	}
}
