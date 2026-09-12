package peeranswers_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const addressOfTheReadDocument = "https://berlin.example/"

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

func answersOfTheReadDocument(t *testing.T) peeranswers.AnsweredQuery {
	t.Helper()

	item := answeredItemMatchingTheWord(t, addressOfTheReadDocument, "berlin")

	return peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachPeerRanking: [][]peeranswers.AnsweredItem{{item}, {item}},
		ItemsInNoOrder:                   []peeranswers.AnsweredItem{item},
		DocumentsHeldPerQueryWord:        map[yacymodel.Hash]int{yacymodel.WordHash("berlin"): 12},
	}
}

func TestEveryItemOfAReadDocumentCarriesWhatItsTextHolds(t *testing.T) {
	t.Parallel()

	answers := answersOfTheReadDocument(t)

	read := answers.CarryingTheTextOfEachDocument(textOfTheReadDocument(t))

	for _, item := range []peeranswers.AnsweredItem{
		read.ItemsInTheOrderOfEachPeerRanking[0][0],
		read.ItemsInTheOrderOfEachPeerRanking[1][0],
		read.ItemsInNoOrder[0],
	} {
		count := item.MatchedWords[yacymodel.WordHash("berlin")]
		if count.Hits != 7 || count.TextWords != 400 ||
			item.Metadata.Snippet != "Berlin holds a wall." {
			t.Fatalf("the item reads %+v, want the counts and the snippet of the text", item)
		}
	}
}

func TestTheDocumentsHeldPerQueryWordStayAsThePeersCountedThem(t *testing.T) {
	t.Parallel()

	answers := answersOfTheReadDocument(t)

	read := answers.CarryingTheTextOfEachDocument(textOfTheReadDocument(t))

	if read.DocumentsHeldPerQueryWord[yacymodel.WordHash("berlin")] != 12 {
		t.Fatalf(
			"the answers hold %+v, want the documents the peers counted",
			read.DocumentsHeldPerQueryWord,
		)
	}
}

func TestAnItemOfADocumentThatWasNotReadStaysAsThePeersAnsweredIt(t *testing.T) {
	t.Parallel()

	answers := peeranswers.AnsweredQuery{
		ItemsInNoOrder: []peeranswers.AnsweredItem{
			answeredItemCountedForTheWord(t, "https://unread.example/", "berlin"),
		},
	}

	read := answers.CarryingTheTextOfEachDocument(textOfTheReadDocument(t))

	count := read.ItemsInNoOrder[0].MatchedWords[yacymodel.WordHash("berlin")]
	if count.Hits != 1 || count.TextWords != 0 {
		t.Fatalf("the item reads %+v, want the count the peers answered", count)
	}
}
