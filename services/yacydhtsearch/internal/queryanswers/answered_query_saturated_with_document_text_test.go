package queryanswers_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const addressOfTheReadDocument = "https://berlin.example/"

func metadataNamedByAddress(t *testing.T, address string) yacymodel.URLMetadata {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return yacymodel.URLMetadata{Hash: hash, Address: address}
}

func foundDocumentCountedForTheWord(
	t *testing.T, address string, word string,
) queryanswers.FoundDocument {
	t.Helper()

	return queryanswers.FoundDocument{
		Metadata: metadataNamedByAddress(t, address),
		MatchedWords: map[yacymodel.Hash]queryanswers.WordCount{
			yacymodel.WordHash(word): {Hits: 1},
		},
	}
}

func documentOf(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	return metadataNamedByAddress(t, address).Hash
}

func textOfTheReadDocument(t *testing.T) map[yacymodel.URLHash]documenttext.DocumentText {
	t.Helper()

	return map[yacymodel.URLHash]documenttext.DocumentText{
		documentOf(t, addressOfTheReadDocument): {
			HitsPerQueryWord: map[yacymodel.Hash]int{yacymodel.WordHash("berlin"): 7},
			AmountOfWords:    400,
			Snippet:          "Berlin holds a wall.",
		},
	}
}

func answersOfTheReadDocument(t *testing.T) queryanswers.AnsweredQuery {
	t.Helper()

	return queryanswers.AnsweredQuery{
		QueryWords: []yacymodel.Hash{yacymodel.WordHash("berlin")},
		FoundDocuments: []queryanswers.FoundDocument{
			foundDocumentCountedForTheWord(t, addressOfTheReadDocument, "berlin"),
		},
		DocumentsHeldPerQueryWord: map[yacymodel.Hash]int{yacymodel.WordHash("berlin"): 12},
	}
}

func TestAReadDocumentCarriesWhatItsTextHolds(t *testing.T) {
	t.Parallel()

	answers := answersOfTheReadDocument(t)

	read := answers.SaturatedWith(textOfTheReadDocument(t))

	foundDocument := read.FoundDocuments[0]
	count := foundDocument.MatchedWords[yacymodel.WordHash("berlin")]
	if count.Hits != 7 || count.TextWords != 400 ||
		foundDocument.Metadata.Snippet != "Berlin holds a wall." {
		t.Fatalf("the found document reads %+v, want the counts and the snippet of the text",
			foundDocument)
	}
}

func TestTheTextOfADocumentCountsAQueryWordNoPeerMatchedItFor(t *testing.T) {
	t.Parallel()

	answers := queryanswers.AnsweredQuery{
		QueryWords: []yacymodel.Hash{
			yacymodel.WordHash("berlin"), yacymodel.WordHash("weather"),
		},
		FoundDocuments: []queryanswers.FoundDocument{
			foundDocumentCountedForTheWord(t, addressOfTheReadDocument, "berlin"),
		},
	}
	textOfTheDocument := map[yacymodel.URLHash]documenttext.DocumentText{
		documentOf(t, addressOfTheReadDocument): {
			HitsPerQueryWord: map[yacymodel.Hash]int{
				yacymodel.WordHash("berlin"):  7,
				yacymodel.WordHash("weather"): 2,
			},
			AmountOfWords: 400,
		},
	}

	read := answers.SaturatedWith(textOfTheDocument)

	count := read.FoundDocuments[0].MatchedWords[yacymodel.WordHash("weather")]
	if count.Hits != 2 || count.TextWords != 400 {
		t.Fatalf(
			"the found document counts %+v for the word no peer matched it for, want the count "+
				"of the text",
			count,
		)
	}
}

func TestTheDocumentsHeldPerQueryWordStayAsThePeersCountedThem(t *testing.T) {
	t.Parallel()

	answers := answersOfTheReadDocument(t)

	read := answers.SaturatedWith(textOfTheReadDocument(t))

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
			foundDocumentCountedForTheWord(t, "https://unread.example/", "berlin"),
		},
	}

	read := answers.SaturatedWith(textOfTheReadDocument(t))

	count := read.FoundDocuments[0].MatchedWords[yacymodel.WordHash("berlin")]
	if count.Hits != 1 || count.TextWords != 0 {
		t.Fatalf("the found document reads %+v, want the count the peers answered", count)
	}
}
