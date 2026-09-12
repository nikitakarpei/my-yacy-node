package relevance_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/relevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type relevanceOfTheGivenDocuments struct {
	relevancePerDocument map[yacymodel.URLHash]float64
}

func (given relevanceOfTheGivenDocuments) RelevancePerDocumentOf(
	_ peeranswers.AnsweredQuery,
) map[yacymodel.URLHash]float64 {
	return given.relevancePerDocument
}

type addressAndItsRelevance struct {
	address   string
	relevance float64
}

func addressesOrderedByRelevance(
	t *testing.T, addressesThePeersPut ...addressAndItsRelevance,
) []string {
	t.Helper()

	itemsOfOnePeerRanking := make([]peeranswers.AnsweredItem, 0, len(addressesThePeersPut))
	relevancePerDocument := map[yacymodel.URLHash]float64{}
	for _, addressAndItsRelevance := range addressesThePeersPut {
		hash, err := yacymodel.URLHashOf(addressAndItsRelevance.address)
		if err != nil {
			t.Fatalf("URLHashOf(%q): %v", addressAndItsRelevance.address, err)
		}
		itemsOfOnePeerRanking = append(itemsOfOnePeerRanking, peeranswers.AnsweredItem{
			Metadata: yacymodel.URLMetadata{
				Hash:    hash,
				Address: addressAndItsRelevance.address,
			},
		})
		relevancePerDocument[hash] = addressAndItsRelevance.relevance
	}

	orderedItems := relevance.New(
		relevanceOfTheGivenDocuments{relevancePerDocument: relevancePerDocument},
	).OrderedItemsOf(peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachPeerRanking: [][]peeranswers.AnsweredItem{itemsOfOnePeerRanking},
	})

	orderedAddresses := make([]string, 0, len(orderedItems))
	for _, orderedItem := range orderedItems {
		orderedAddresses = append(orderedAddresses, orderedItem.Metadata.Address)
	}

	return orderedAddresses
}

func TestTheMostRelevantDocumentComesFirst(t *testing.T) {
	t.Parallel()

	got := addressesOrderedByRelevance(
		t,
		addressAndItsRelevance{address: "https://third.example/", relevance: 1},
		addressAndItsRelevance{address: "https://first.example/", relevance: 10},
		addressAndItsRelevance{address: "https://second.example/", relevance: 5},
	)

	want := []string{
		"https://first.example/",
		"https://second.example/",
		"https://third.example/",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestDocumentsOfEqualRelevanceKeepTheOrderThePeersPutThem(t *testing.T) {
	t.Parallel()

	got := addressesOrderedByRelevance(
		t,
		addressAndItsRelevance{address: "https://a.example/", relevance: 5},
		addressAndItsRelevance{address: "https://b.example/", relevance: 5},
		addressAndItsRelevance{address: "https://c.example/", relevance: 5},
	)

	want := []string{"https://a.example/", "https://b.example/", "https://c.example/"}
	if !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want the order the peers put %v", got, want)
	}
}

func TestNoAnsweredItemMakesNoOrderedItem(t *testing.T) {
	t.Parallel()

	if got := addressesOrderedByRelevance(t); len(got) != 0 {
		t.Fatalf("the relevance order reads %v, want no item", got)
	}
}
