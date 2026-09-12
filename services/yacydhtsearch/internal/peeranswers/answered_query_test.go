package peeranswers_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func metadataNamedByAddress(t *testing.T, address string) yacymodel.URLMetadata {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return yacymodel.URLMetadata{Hash: hash, Address: address}
}

func answeredItemAt(t *testing.T, address string) peeranswers.AnsweredItem {
	t.Helper()

	return peeranswers.AnsweredItem{Metadata: metadataNamedByAddress(t, address)}
}

func answeredItemCountedForTheWord(
	t *testing.T, address string, word string,
) peeranswers.AnsweredItem {
	t.Helper()

	return peeranswers.AnsweredItem{
		Metadata: metadataNamedByAddress(t, address),
		MatchedWords: map[yacymodel.Hash]peeranswers.WordCount{
			yacymodel.WordHash(word): {Hits: 1},
		},
	}
}

func answeredItemMatchingTheWord(
	t *testing.T, address string, word string,
) peeranswers.AnsweredItem {
	t.Helper()

	return answeredItemAt(t, address).
		MatchingTheWords([]yacymodel.Hash{yacymodel.WordHash(word)})
}

func addressesOfAnsweredItems(answeredItems []peeranswers.AnsweredItem) []string {
	addresses := make([]string, 0, len(answeredItems))
	for _, answeredItem := range answeredItems {
		addresses = append(addresses, answeredItem.Metadata.Address)
	}

	return addresses
}

func TestOneItemOfEachPeerComesBackPerRound(t *testing.T) {
	t.Parallel()

	answers := peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachPeerRanking: [][]peeranswers.AnsweredItem{
			{answeredItemAt(t, "https://a.example/1"), answeredItemAt(t, "https://a.example/2")},
			{answeredItemAt(t, "https://b.example/1")},
		},
	}

	items := answers.ItemOfEachAnsweredDocument()

	want := []string{"https://a.example/1", "https://b.example/1", "https://a.example/2"}
	if got := addressesOfAnsweredItems(items); !slices.Equal(got, want) {
		t.Fatalf("ItemOfEachAnsweredDocument = %v, want %v", got, want)
	}
}

func TestOneURLComesBackOnceHoweverManyPeersAnsweredIt(t *testing.T) {
	t.Parallel()

	shared := answeredItemAt(t, "https://shared.example/")
	answers := peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachPeerRanking: [][]peeranswers.AnsweredItem{
			{shared}, {shared},
		},
	}

	items := answers.ItemOfEachAnsweredDocument()

	if got := addressesOfAnsweredItems(items); !slices.Equal(
		got, []string{"https://shared.example/"},
	) {
		t.Fatalf("ItemOfEachAnsweredDocument = %v, want the shared address once", got)
	}
}

func TestTheCountsOfEveryPeerThatAnsweredADocumentComeBackTogether(t *testing.T) {
	t.Parallel()

	address := "https://shared.example/"
	answers := peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachPeerRanking: [][]peeranswers.AnsweredItem{
			{answeredItemMatchingTheWord(t, address, "berlin")},
			{answeredItemCountedForTheWord(t, address, "berlin")},
			{answeredItemCountedForTheWord(t, address, "weather")},
		},
	}

	items := answers.ItemOfEachAnsweredDocument()

	if len(items) != 1 {
		t.Fatalf("ItemOfEachAnsweredDocument = %v, want the shared document once", items)
	}
	matchedWords := items[0].MatchedWords
	if !matchedWords[yacymodel.WordHash("berlin")].CountedByAPeer() ||
		!matchedWords[yacymodel.WordHash("weather")].CountedByAPeer() {
		t.Fatalf("the document matched %v, want the counts of every peer that answered it",
			matchedWords)
	}
}

func TestAnItemOfNoOrderComesBackBehindTheItemsThePeersPutInOrder(t *testing.T) {
	t.Parallel()

	answers := peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachPeerRanking: [][]peeranswers.AnsweredItem{
			{answeredItemAt(t, "https://ordered.example/")},
		},
		ItemsInNoOrder: []peeranswers.AnsweredItem{
			answeredItemAt(t, "https://unordered.example/"),
		},
	}

	items := answers.ItemOfEachAnsweredDocument()

	want := []string{"https://ordered.example/", "https://unordered.example/"}
	if got := addressesOfAnsweredItems(items); !slices.Equal(got, want) {
		t.Fatalf("ItemOfEachAnsweredDocument = %v, want %v", got, want)
	}
}

func TestAnItemOfNoOrderCarriesItsCountsToTheDocumentAPeerAlsoAnswered(t *testing.T) {
	t.Parallel()

	address := "https://shared.example/"
	answers := peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachPeerRanking: [][]peeranswers.AnsweredItem{
			{answeredItemMatchingTheWord(t, address, "berlin")},
		},
		ItemsInNoOrder: []peeranswers.AnsweredItem{
			answeredItemCountedForTheWord(t, address, "berlin"),
		},
	}

	items := answers.ItemOfEachAnsweredDocument()

	if len(items) != 1 ||
		!items[0].MatchedWords[yacymodel.WordHash("berlin")].CountedByAPeer() {
		t.Fatalf("ItemOfEachAnsweredDocument = %v, want the document once with its count", items)
	}
}

func TestNoItemComesBackWhenNoPeerAnswered(t *testing.T) {
	t.Parallel()

	if items := (peeranswers.AnsweredQuery{}).ItemOfEachAnsweredDocument(); len(items) != 0 {
		t.Fatalf("ItemOfEachAnsweredDocument = %+v, want no item", items)
	}
}
