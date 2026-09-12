package peerorder_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/peerorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func itemAt(t *testing.T, address string) peeranswers.AnsweredItem {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return peeranswers.AnsweredItem{
		Metadata: yacymodel.URLMetadata{Hash: hash, Address: address},
	}
}

func addressesOrderedBy(answers peeranswers.AnsweredQuery) []string {
	orderedItems := peerorder.Ordering{}.OrderedItemsOf(answers)

	addresses := make([]string, 0, len(orderedItems))
	for _, orderedItem := range orderedItems {
		addresses = append(addresses, orderedItem.Metadata.Address)
	}

	return addresses
}

func TestTheItemsComeBackOneOfEachPeerPerRound(t *testing.T) {
	t.Parallel()

	answers := peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachPeerRanking: [][]peeranswers.AnsweredItem{
			{itemAt(t, "https://a.example/1"), itemAt(t, "https://a.example/2")},
			{itemAt(t, "https://b.example/1")},
		},
	}

	want := []string{"https://a.example/1", "https://b.example/1", "https://a.example/2"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the peer order reads %v, want %v", got, want)
	}
}

func TestTheItemsInNoOrderComeBackBehindTheItemsThePeersPutInOrder(t *testing.T) {
	t.Parallel()

	answers := peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachPeerRanking: [][]peeranswers.AnsweredItem{
			{itemAt(t, "https://ordered.example/")},
		},
		ItemsInNoOrder: []peeranswers.AnsweredItem{
			itemAt(t, "https://unordered.example/"),
		},
	}

	want := []string{"https://ordered.example/", "https://unordered.example/"}
	if got := addressesOrderedBy(answers); !slices.Equal(got, want) {
		t.Fatalf("the peer order reads %v, want %v", got, want)
	}
}
