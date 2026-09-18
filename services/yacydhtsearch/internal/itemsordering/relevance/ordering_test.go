package relevance_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/relevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type relevanceOfTheGivenDocuments struct {
	relevancePerDocument map[yacymodel.URLHash]float64
}

func (given relevanceOfTheGivenDocuments) RelevancePerDocumentOf(
	_ queryanswers.AnsweredQuery,
) map[yacymodel.URLHash]float64 {
	return given.relevancePerDocument
}

type addressAndItsRelevance struct {
	address   string
	relevance float64
}

func addressesOrderedByRelevance(
	t *testing.T, addressesInTheFoundOrder ...addressAndItsRelevance,
) []string {
	t.Helper()

	foundDocuments := make([]queryanswers.FoundDocument, 0, len(addressesInTheFoundOrder))
	relevancePerDocument := map[yacymodel.URLHash]float64{}
	for _, addressAndItsRelevance := range addressesInTheFoundOrder {
		hash, err := yacymodel.URLHashOf(addressAndItsRelevance.address)
		if err != nil {
			t.Fatalf("URLHashOf(%q): %v", addressAndItsRelevance.address, err)
		}
		foundDocuments = append(foundDocuments, queryanswers.FoundDocument{
			Metadata: yacymodel.URLMetadata{
				Hash:    hash,
				Address: addressAndItsRelevance.address,
			},
		})
		relevancePerDocument[hash] = addressAndItsRelevance.relevance
	}

	orderedItems := relevance.New(
		relevanceOfTheGivenDocuments{relevancePerDocument: relevancePerDocument},
	).OrderedItemsOf(queryanswers.AnsweredQuery{
		FoundDocuments: foundDocuments,
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

func TestDocumentsOfEqualRelevanceKeepTheOrderTheyWereFoundIn(t *testing.T) {
	t.Parallel()

	got := addressesOrderedByRelevance(
		t,
		addressAndItsRelevance{address: "https://a.example/", relevance: 5},
		addressAndItsRelevance{address: "https://b.example/", relevance: 5},
		addressAndItsRelevance{address: "https://c.example/", relevance: 5},
	)

	want := []string{"https://a.example/", "https://b.example/", "https://c.example/"}
	if !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want the order they were found in %v", got, want)
	}
}

func TestNoFoundDocumentMakesNoOrderedItem(t *testing.T) {
	t.Parallel()

	if got := addressesOrderedByRelevance(t); len(got) != 0 {
		t.Fatalf("the relevance order reads %v, want no item", got)
	}
}

func TestOrderingLeavesTheFoundDocumentsOfTheAnswersInTheirOrder(t *testing.T) {
	t.Parallel()

	answers := queryanswers.AnsweredQuery{
		FoundDocuments: []queryanswers.FoundDocument{
			foundDocumentAt(t, "https://less.example/"),
			foundDocumentAt(t, "https://more.example/"),
		},
	}

	relevance.New(relevanceByFoundPlace{}).OrderedItemsOf(answers)

	if answers.FoundDocuments[0].Metadata.Address != "https://less.example/" {
		t.Fatalf("the answers read %v after ordering, want the order they were found in",
			answers.FoundDocuments)
	}
}

func foundDocumentAt(t *testing.T, address string) queryanswers.FoundDocument {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return queryanswers.FoundDocument{Metadata: yacymodel.URLMetadata{Hash: hash, Address: address}}
}

type relevanceByFoundPlace struct{}

func (relevanceByFoundPlace) RelevancePerDocumentOf(
	answers queryanswers.AnsweredQuery,
) map[yacymodel.URLHash]float64 {
	relevancePerDocument := map[yacymodel.URLHash]float64{}
	for place, foundDocument := range answers.FoundDocuments {
		relevancePerDocument[foundDocument.Metadata.Hash] = float64(place)
	}

	return relevancePerDocument
}
