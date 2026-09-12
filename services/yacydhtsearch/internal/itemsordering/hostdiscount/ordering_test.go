package hostdiscount_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/hostdiscount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type orderingOfTheGivenRelevancePerAddress struct {
	itemsInFallingOrderOfRelevance []peeranswers.AnsweredItem
	relevancePerDocument           map[yacymodel.URLHash]float64
}

func (o orderingOfTheGivenRelevancePerAddress) OrderedItemsOf(
	_ peeranswers.AnsweredQuery,
) []peeranswers.AnsweredItem {
	return o.itemsInFallingOrderOfRelevance
}

func (o orderingOfTheGivenRelevancePerAddress) RelevancePerDocumentOf(
	_ peeranswers.AnsweredQuery,
) map[yacymodel.URLHash]float64 {
	return o.relevancePerDocument
}

type addressAndItsRelevance struct {
	address   string
	relevance float64
}

func addressesOrderedWithTheHostDiscount(
	t *testing.T, addressesInFallingOrderOfRelevance ...addressAndItsRelevance,
) []string {
	t.Helper()

	ordering := orderingOfTheGivenRelevancePerAddress{
		relevancePerDocument: map[yacymodel.URLHash]float64{},
	}
	for _, addressAndItsRelevance := range addressesInFallingOrderOfRelevance {
		hash, err := yacymodel.URLHashOf(addressAndItsRelevance.address)
		if err != nil {
			t.Fatalf("URLHashOf(%q): %v", addressAndItsRelevance.address, err)
		}
		ordering.itemsInFallingOrderOfRelevance = append(
			ordering.itemsInFallingOrderOfRelevance,
			peeranswers.AnsweredItem{
				Metadata: yacymodel.URLMetadata{
					Hash:    hash,
					Address: addressAndItsRelevance.address,
				},
			},
		)
		ordering.relevancePerDocument[hash] = addressAndItsRelevance.relevance
	}

	orderedItems := hostdiscount.New(ordering).OrderedItemsOf(peeranswers.AnsweredQuery{})
	orderedAddresses := make([]string, 0, len(orderedItems))
	for _, orderedItem := range orderedItems {
		orderedAddresses = append(orderedAddresses, orderedItem.Metadata.Address)
	}

	return orderedAddresses
}

func TestASecondItemOfAHostWaitsForEveryItemTheDiscountLeavesAboveIt(t *testing.T) {
	t.Parallel()

	got := addressesOrderedWithTheHostDiscount(
		t,
		addressAndItsRelevance{address: "https://one.example/a", relevance: 10},
		addressAndItsRelevance{address: "https://one.example/b", relevance: 8},
		addressAndItsRelevance{address: "https://two.example/a", relevance: 6},
		addressAndItsRelevance{address: "https://three.example/a", relevance: 3},
	)

	want := []string{
		"https://one.example/a",
		"https://two.example/a",
		"https://one.example/b",
		"https://three.example/a",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the host discount order reads %v, want %v", got, want)
	}
}

func TestAStrongSecondItemOfAHostComesBeforeAWeakerItemOfAnotherHost(t *testing.T) {
	t.Parallel()

	got := addressesOrderedWithTheHostDiscount(
		t,
		addressAndItsRelevance{address: "https://one.example/a", relevance: 10},
		addressAndItsRelevance{address: "https://one.example/b", relevance: 9},
		addressAndItsRelevance{address: "https://two.example/a", relevance: 4},
	)

	want := []string{
		"https://one.example/a",
		"https://one.example/b",
		"https://two.example/a",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the host discount order reads %v, want %v", got, want)
	}
}

func TestEachFurtherItemOfOneHostTakesAFurtherDiscount(t *testing.T) {
	t.Parallel()

	got := addressesOrderedWithTheHostDiscount(
		t,
		addressAndItsRelevance{address: "https://one.example/a", relevance: 10},
		addressAndItsRelevance{address: "https://one.example/b", relevance: 10},
		addressAndItsRelevance{address: "https://one.example/c", relevance: 10},
		addressAndItsRelevance{address: "https://two.example/a", relevance: 4},
		addressAndItsRelevance{address: "https://two.example/b", relevance: 4},
	)

	want := []string{
		"https://one.example/a",
		"https://one.example/b",
		"https://two.example/a",
		"https://one.example/c",
		"https://two.example/b",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the host discount order reads %v, want %v", got, want)
	}
}

func TestTheItemsOfEqualDiscountedRelevanceKeepTheOrderTheWrappedOrderingPutThem(t *testing.T) {
	t.Parallel()

	got := addressesOrderedWithTheHostDiscount(
		t,
		addressAndItsRelevance{address: "https://one.example/a", relevance: 5},
		addressAndItsRelevance{address: "https://two.example/a", relevance: 5},
		addressAndItsRelevance{address: "https://three.example/a", relevance: 5},
	)

	want := []string{
		"https://one.example/a",
		"https://two.example/a",
		"https://three.example/a",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the host discount order reads %v, want %v", got, want)
	}
}

func TestOnePathOfAHostDoesNotMakeAnotherHost(t *testing.T) {
	t.Parallel()

	got := addressesOrderedWithTheHostDiscount(
		t,
		addressAndItsRelevance{address: "https://one.example/a", relevance: 10},
		addressAndItsRelevance{address: "http://one.example:8080/b", relevance: 8},
		addressAndItsRelevance{address: "https://two.example/a", relevance: 6},
	)

	want := []string{
		"https://one.example/a",
		"https://two.example/a",
		"http://one.example:8080/b",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the host discount order reads %v, want %v", got, want)
	}
}

func TestAnAddressThatHoldsNoHostIsItsOwnHost(t *testing.T) {
	t.Parallel()

	got := addressesOrderedWithTheHostDiscount(
		t,
		addressAndItsRelevance{address: "documents/a", relevance: 10},
		addressAndItsRelevance{address: "documents/b", relevance: 9},
		addressAndItsRelevance{address: "https://one.example/a", relevance: 6},
	)

	want := []string{
		"documents/a",
		"documents/b",
		"https://one.example/a",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the host discount order reads %v, want %v", got, want)
	}
}

func TestNoAnsweredItemMakesNoOrderedItem(t *testing.T) {
	t.Parallel()

	if got := addressesOrderedWithTheHostDiscount(t); len(got) != 0 {
		t.Fatalf("the host discount order reads %v, want no item", got)
	}
}
