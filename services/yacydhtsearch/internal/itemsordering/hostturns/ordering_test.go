package hostturns_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/hostturns"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type orderingOfTheGivenAddresses struct {
	addresses []string
}

func (o orderingOfTheGivenAddresses) OrderedItemsOf(
	_ peeranswers.AnsweredQuery,
) []peeranswers.AnsweredItem {
	items := make([]peeranswers.AnsweredItem, 0, len(o.addresses))
	for _, address := range o.addresses {
		items = append(items, peeranswers.AnsweredItem{
			Metadata: yacymodel.URLMetadata{Address: address},
		})
	}

	return items
}

func addressesOrderedInTurns(addresses ...string) []string {
	orderedItems := hostturns.New(orderingOfTheGivenAddresses{addresses: addresses}).
		OrderedItemsOf(peeranswers.AnsweredQuery{})

	orderedAddresses := make([]string, 0, len(orderedItems))
	for _, orderedItem := range orderedItems {
		orderedAddresses = append(orderedAddresses, orderedItem.Metadata.Address)
	}

	return orderedAddresses
}

func TestTheFirstItemOfEachHostComesBeforeTheSecondItemOfAnyHost(t *testing.T) {
	t.Parallel()

	got := addressesOrderedInTurns(
		"https://one.example/a",
		"https://one.example/b",
		"https://two.example/a",
		"https://three.example/a",
	)

	want := []string{
		"https://one.example/a",
		"https://two.example/a",
		"https://three.example/a",
		"https://one.example/b",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the host turns order reads %v, want %v", got, want)
	}
}

func TestTheItemsOfOneTurnKeepTheOrderTheWrappedOrderingPutThem(t *testing.T) {
	t.Parallel()

	got := addressesOrderedInTurns(
		"https://one.example/a",
		"https://two.example/a",
		"https://three.example/a",
	)

	want := []string{
		"https://one.example/a",
		"https://two.example/a",
		"https://three.example/a",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the host turns order reads %v, want %v", got, want)
	}
}

func TestEachFurtherItemOfOneHostWaitsForAFurtherTurn(t *testing.T) {
	t.Parallel()

	got := addressesOrderedInTurns(
		"https://one.example/a",
		"https://one.example/b",
		"https://one.example/c",
		"https://two.example/a",
		"https://two.example/b",
	)

	want := []string{
		"https://one.example/a",
		"https://two.example/a",
		"https://one.example/b",
		"https://two.example/b",
		"https://one.example/c",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the host turns order reads %v, want %v", got, want)
	}
}

func TestOnePathOfAHostDoesNotMakeAnotherHost(t *testing.T) {
	t.Parallel()

	got := addressesOrderedInTurns(
		"https://one.example/a",
		"http://one.example:8080/b",
		"https://two.example/a",
	)

	want := []string{
		"https://one.example/a",
		"https://two.example/a",
		"http://one.example:8080/b",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the host turns order reads %v, want %v", got, want)
	}
}

func TestAnAddressNoNodeCanReadIsItsOwnHost(t *testing.T) {
	t.Parallel()

	got := addressesOrderedInTurns(
		"https://berlin weather.example/a",
		"https://berlin weather.example/b",
		"https://one.example/a",
		"https://one.example/b",
	)

	want := []string{
		"https://berlin weather.example/a",
		"https://berlin weather.example/b",
		"https://one.example/a",
		"https://one.example/b",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the host turns order reads %v, want %v", got, want)
	}
}

func TestNoAnsweredItemMakesNoOrderedItem(t *testing.T) {
	t.Parallel()

	if got := addressesOrderedInTurns(); len(got) != 0 {
		t.Fatalf("the host turns order reads %v, want no item", got)
	}
}
