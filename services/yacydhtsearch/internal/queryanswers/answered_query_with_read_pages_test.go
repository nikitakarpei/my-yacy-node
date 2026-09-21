package queryanswers_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const addressOfReadDocument = "https://berlin.example/"

func documentOf(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return hash
}

func answersOfDocumentAPeerMatchedForWord(
	t *testing.T, address string, word string,
) queryanswers.AnsweredQuery {
	t.Helper()

	return queryanswers.AnsweredQuery{
		QueryWords: []yacymodel.Hash{yacymodel.WordHash(word)},
		FoundDocuments: []queryanswers.FoundDocument{queryanswers.FoundDocumentOf(
			documentOf(t, address),
			[]queryanswers.MetadataReplica{{Metadata: yacymodel.URLMetadata{
				Address: address, Title: "The title a peer sent",
			}}},
			[]queryanswers.PostingReplica{{
				Word:    yacymodel.Some(yacymodel.WordHash(word)),
				Posting: yacymodel.RWIPosting{Hits: 1, LocalLinks: 2},
			}},
		)},
		DocumentsHeldPerQueryWord: map[yacymodel.Hash]int{yacymodel.WordHash(word): 12},
	}
}

func answersOfReadDocument(t *testing.T) queryanswers.AnsweredQuery {
	t.Helper()

	return answersOfDocumentAPeerMatchedForWord(t, addressOfReadDocument, "berlin")
}

func pageContentsOfReadDocument(t *testing.T) map[yacymodel.URLHash]pagecontents.PageContents {
	t.Helper()

	return map[yacymodel.URLHash]pagecontents.PageContents{
		documentOf(t, addressOfReadDocument): {
			Title:            "Berlin",
			HitsPerQueryWord: map[yacymodel.Hash]int{yacymodel.WordHash("berlin"): 7},
			AmountOfWords:    400,
			Snippet:          "Berlin holds a wall.",
			LinkCounts:       pagecontents.LinkCounts{LocalLinks: 25, ExternalLinks: 4},
		},
	}
}

func TestReadDocumentCarriesWhatItsTextHolds(t *testing.T) {
	t.Parallel()

	answers := answersOfReadDocument(t)

	read := answers.WithReadPages(pageContentsOfReadDocument(t))

	foundDocument := read.FoundDocuments[0]
	if foundDocument.Facts.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 7 ||
		foundDocument.Facts.AmountOfWords.OrElse(0) != 400 ||
		foundDocument.Snippet != "Berlin holds a wall." ||
		foundDocument.Title != "Berlin" {
		t.Fatalf(
			"the document reads %+v, want the counts, the snippet and the title of the text",
			foundDocument,
		)
	}
}

func TestTextOfDocumentCountsQueryWordNoPeerMatchedItFor(t *testing.T) {
	t.Parallel()

	answers := answersOfReadDocument(t)
	answers.QueryWords = append(answers.QueryWords, yacymodel.WordHash("weather"))
	pageContentsOfDocument := pageContentsOfReadDocument(t)
	pageContents := pageContentsOfDocument[documentOf(t, addressOfReadDocument)]
	pageContents.HitsPerQueryWord[yacymodel.WordHash("weather")] = 2
	pageContentsOfDocument[documentOf(t, addressOfReadDocument)] = pageContents

	read := answers.WithReadPages(pageContentsOfDocument)

	facts := read.FoundDocuments[0].Facts
	if facts.HitsPerQueryWord[yacymodel.WordHash("weather")] != 2 ||
		facts.AmountOfWords.OrElse(0) != 400 {
		t.Fatalf(
			"the document reads %+v for the word no peer matched it for, want the count of the "+
				"text",
			facts,
		)
	}
}

func TestReadPageWithoutTitleKeepsTitleAPeerSent(t *testing.T) {
	t.Parallel()

	answers := answersOfReadDocument(t)
	pageContentsWithoutTitle := pageContentsOfReadDocument(t)
	pageContents := pageContentsWithoutTitle[documentOf(t, addressOfReadDocument)]
	pageContents.Title = ""
	pageContentsWithoutTitle[documentOf(t, addressOfReadDocument)] = pageContents

	read := answers.WithReadPages(pageContentsWithoutTitle)

	if read.FoundDocuments[0].Title != "The title a peer sent" {
		t.Fatalf(
			"the found document holds the title %q, want the title the peer sent",
			read.FoundDocuments[0].Title,
		)
	}
}

func TestDocumentsHeldPerQueryWordStayAsThePeersCountedThem(t *testing.T) {
	t.Parallel()

	answers := answersOfReadDocument(t)

	read := answers.WithReadPages(pageContentsOfReadDocument(t))

	if read.DocumentsHeldPerQueryWord[yacymodel.WordHash("berlin")] != 12 {
		t.Fatalf(
			"the answers hold %+v, want the documents the peers counted",
			read.DocumentsHeldPerQueryWord,
		)
	}
}

func TestDocumentThatWasNotReadStaysAsThePeersCountedIt(t *testing.T) {
	t.Parallel()

	answers := answersOfDocumentAPeerMatchedForWord(t, "https://unread.example/", "berlin")

	read := answers.WithReadPages(pageContentsOfReadDocument(t))

	facts := read.FoundDocuments[0].Facts
	if facts.HitsPerQueryWord[yacymodel.WordHash("berlin")] != 1 ||
		facts.AmountOfWords.Present() ||
		facts.AmountOfLinks.OrElse(0) != 2 {
		t.Fatalf("the document reads %+v, want the counts the peers answered", facts)
	}
}

func TestReadPageThatMovedGivesItsDocumentAddressItMovedTo(t *testing.T) {
	t.Parallel()

	answers := answersOfReadDocument(t)
	pageContentsOfMovedPage := pageContentsOfReadDocument(t)
	pageContents := pageContentsOfMovedPage[documentOf(t, addressOfReadDocument)]
	pageContents.Address = "https://berlin.example/moved"
	pageContentsOfMovedPage[documentOf(t, addressOfReadDocument)] = pageContents

	read := answers.WithReadPages(pageContentsOfMovedPage)

	foundDocument := read.FoundDocuments[0]
	if foundDocument.Address != "https://berlin.example/moved" ||
		foundDocument.Hash != documentOf(t, addressOfReadDocument) {
		t.Fatalf(
			"the found document reads %+v, want the address the page moved to under its own hash",
			foundDocument,
		)
	}
}

func TestReadPageThatDidNotMoveKeepsAddressAPeerSent(t *testing.T) {
	t.Parallel()

	read := answersOfReadDocument(t).WithReadPages(pageContentsOfReadDocument(t))

	if read.FoundDocuments[0].Address != addressOfReadDocument {
		t.Fatalf(
			"the found document holds the address %q, want the address the peer sent",
			read.FoundDocuments[0].Address,
		)
	}
}

func TestReadPageGivesItsDocumentLinksItHoldsInPlaceOfLinksAPeerCounted(t *testing.T) {
	t.Parallel()

	answers := answersOfReadDocument(t)

	read := answers.WithReadPages(pageContentsOfReadDocument(t))

	facts := read.FoundDocuments[0].Facts
	if amountOfLinks, counted := facts.AmountOfLinks.Get(); !counted || amountOfLinks != 29 {
		t.Fatalf(
			"the document holds %d links counted %t, want the 29 links the page holds",
			amountOfLinks,
			counted,
		)
	}
}

func TestReadPageCountsQueryWordItHoldsNoneOfInPlaceOfPeerThatMatchedIt(t *testing.T) {
	t.Parallel()

	answers := answersOfDocumentAPeerMatchedForWord(t, addressOfReadDocument, "weather")

	read := answers.WithReadPages(pageContentsOfReadDocument(t))

	facts := read.FoundDocuments[0].Facts
	if facts.HitsPerQueryWord[yacymodel.WordHash("weather")] != 0 {
		t.Fatalf(
			"the document counts %d hits of the query word its read page holds none of, want the "+
				"count of the text",
			facts.HitsPerQueryWord[yacymodel.WordHash("weather")],
		)
	}
}

func TestOnlyAReadPageCountsQueryPhrases(t *testing.T) {
	t.Parallel()

	answers := answersOfReadDocument(t)
	unread := answersOfDocumentAPeerMatchedForWord(t, "https://unread.example/", "berlin")
	answers.FoundDocuments = append(answers.FoundDocuments, unread.FoundDocuments[0])
	pageContentsOfDocument := pageContentsOfReadDocument(t)
	pageContents := pageContentsOfDocument[documentOf(t, addressOfReadDocument)]
	pageContents.QueryPhraseHits = 3
	pageContentsOfDocument[documentOf(t, addressOfReadDocument)] = pageContents

	read := answers.WithReadPages(pageContentsOfDocument)

	queryPhraseHitsOfReadDocument := read.FoundDocuments[0].Facts.QueryPhraseHits
	queryPhraseHitsOfUnreadDocument := read.FoundDocuments[1].Facts.QueryPhraseHits
	if queryPhraseHitsOfReadDocument.OrElse(0) != 3 ||
		queryPhraseHitsOfUnreadDocument.Present() {
		t.Fatalf(
			"the read document holds %+v query phrase hits and the unread one %+v, want the "+
				"three hits of the read page and none for the unread document",
			queryPhraseHitsOfReadDocument,
			queryPhraseHitsOfUnreadDocument,
		)
	}
}
