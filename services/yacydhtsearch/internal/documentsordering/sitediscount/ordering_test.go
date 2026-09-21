package sitediscount_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentsordering/sitediscount"
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

func addressesOrderedWithTheSiteDiscount(
	t *testing.T, addressesInFallingOrderOfRelevance ...addressAndItsRelevance,
) []string {
	t.Helper()

	foundDocuments := make(
		[]queryanswers.FoundDocument, 0, len(addressesInFallingOrderOfRelevance),
	)
	relevancePerDocument := map[yacymodel.URLHash]float64{}
	for _, addressAndItsRelevance := range addressesInFallingOrderOfRelevance {
		hash, err := yacymodel.URLHashOf(addressAndItsRelevance.address)
		if err != nil {
			t.Fatalf("URLHashOf(%q): %v", addressAndItsRelevance.address, err)
		}
		foundDocuments = append(foundDocuments, queryanswers.FoundDocument{
			Hash:    hash,
			Address: addressAndItsRelevance.address,
		})
		relevancePerDocument[hash] = addressAndItsRelevance.relevance
	}

	orderedDocuments := sitediscount.New(
		relevanceOfTheGivenDocuments{relevancePerDocument: relevancePerDocument},
	).OrderedDocumentsOf(queryanswers.AnsweredQuery{
		FoundDocuments: foundDocuments,
	})
	orderedAddresses := make([]string, 0, len(orderedDocuments))
	for _, orderedDocument := range orderedDocuments {
		orderedAddresses = append(orderedAddresses, orderedDocument.Address)
	}

	return orderedAddresses
}

func TestASecondItemOfASiteWaitsForEveryItemTheDiscountLeavesAboveIt(t *testing.T) {
	t.Parallel()

	got := addressesOrderedWithTheSiteDiscount(
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
		t.Fatalf("the site discount order reads %v, want %v", got, want)
	}
}

func TestAStrongSecondItemOfASiteComesBeforeAWeakerItemOfAnotherSite(t *testing.T) {
	t.Parallel()

	got := addressesOrderedWithTheSiteDiscount(
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
		t.Fatalf("the site discount order reads %v, want %v", got, want)
	}
}

func TestEachFurtherItemOfOneSiteTakesAFurtherDiscount(t *testing.T) {
	t.Parallel()

	got := addressesOrderedWithTheSiteDiscount(
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
		t.Fatalf("the site discount order reads %v, want %v", got, want)
	}
}

func TestTheItemsOfEqualDiscountedRelevanceKeepTheOrderTheyWereFoundIn(t *testing.T) {
	t.Parallel()

	got := addressesOrderedWithTheSiteDiscount(
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
		t.Fatalf("the site discount order reads %v, want %v", got, want)
	}
}

func TestOnePathOfASiteDoesNotMakeAnotherSite(t *testing.T) {
	t.Parallel()

	got := addressesOrderedWithTheSiteDiscount(
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
		t.Fatalf("the site discount order reads %v, want %v", got, want)
	}
}

func TestTheWorldWideWebLabelDoesNotMakeAnotherSite(t *testing.T) {
	t.Parallel()

	got := addressesOrderedWithTheSiteDiscount(
		t,
		addressAndItsRelevance{address: "https://www.one.example/a", relevance: 10},
		addressAndItsRelevance{address: "https://one.example/b", relevance: 8},
		addressAndItsRelevance{address: "https://two.example/a", relevance: 6},
	)

	want := []string{
		"https://www.one.example/a",
		"https://two.example/a",
		"https://one.example/b",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the site discount order reads %v, want %v", got, want)
	}
}

func TestAnAddressThatHoldsNoHostIsItsOwnSite(t *testing.T) {
	t.Parallel()

	got := addressesOrderedWithTheSiteDiscount(
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
		t.Fatalf("the site discount order reads %v, want %v", got, want)
	}
}

func TestNoFoundDocumentMakesNoOrderedItem(t *testing.T) {
	t.Parallel()

	if got := addressesOrderedWithTheSiteDiscount(t); len(got) != 0 {
		t.Fatalf("the site discount order reads %v, want no foundDocument", got)
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

	sitediscount.New(relevanceByFoundPlace{}).OrderedDocumentsOf(answers)

	if answers.FoundDocuments[0].Address != "https://less.example/" {
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

	return queryanswers.FoundDocument{Hash: hash, Address: address}
}

type relevanceByFoundPlace struct{}

func (relevanceByFoundPlace) RelevancePerDocumentOf(
	answers queryanswers.AnsweredQuery,
) map[yacymodel.URLHash]float64 {
	relevancePerDocument := map[yacymodel.URLHash]float64{}
	for place, foundDocument := range answers.FoundDocuments {
		relevancePerDocument[foundDocument.Hash] = float64(place)
	}

	return relevancePerDocument
}

func TestADocumentBelowNoRelevanceDoesNotRiseWhenItsSiteRepeats(t *testing.T) {
	t.Parallel()

	got := addressesOrderedWithTheSiteDiscount(
		t,
		addressAndItsRelevance{address: "https://one.example/a", relevance: 10},
		addressAndItsRelevance{address: "https://one.example/b", relevance: 9},
		addressAndItsRelevance{address: "https://one.example/c", relevance: 8},
		addressAndItsRelevance{address: "https://two.example/a", relevance: -1},
		addressAndItsRelevance{address: "https://one.example/d", relevance: -4},
	)

	want := []string{
		"https://one.example/a",
		"https://one.example/b",
		"https://one.example/c",
		"https://two.example/a",
		"https://one.example/d",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the site discount order reads %v, want %v", got, want)
	}
}
