package searchresult_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func itemAt(t *testing.T, address string) searchresult.Item {
	t.Helper()

	return searchresult.ItemFrom(metadataNamedByAddress(t, address))
}

func metadataNamedByAddress(t *testing.T, address string) yacymodel.URLMetadata {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return yacymodel.URLMetadata{Hash: hash, Address: address}
}

func addressesOf(items []searchresult.Item) []string {
	addresses := make([]string, 0, len(items))
	for _, item := range items {
		addresses = append(addresses, item.Address)
	}

	return addresses
}

func rankingOver(t *testing.T, addresses ...string) searchresult.Ranking {
	t.Helper()

	items := make([]searchresult.Item, 0, len(addresses))
	for _, address := range addresses {
		items = append(items, itemAt(t, address))
	}

	return searchresult.Ranking{Items: items}
}

func TestAPageCarriesTheItemsItStartsAt(t *testing.T) {
	t.Parallel()

	ranking := rankingOver(t, "https://a.example/1", "https://a.example/2", "https://a.example/3")

	page := ranking.PageFrom(1, 2)

	want := []string{"https://a.example/2", "https://a.example/3"}
	if got := addressesOf(page.Items); !slices.Equal(got, want) {
		t.Fatalf("PageFrom(1, 2) = %v, want %v", got, want)
	}
}

func TestAPageStopsAtTheLastItemTheRankingHolds(t *testing.T) {
	t.Parallel()

	ranking := rankingOver(t, "https://a.example/1", "https://a.example/2")

	if page := ranking.PageFrom(1, 10); len(page.Items) != 1 {
		t.Fatalf("PageFrom(1, 10) carried %d items, want the last one", len(page.Items))
	}
}

func TestAPagePastTheLastItemCarriesNothing(t *testing.T) {
	t.Parallel()

	ranking := rankingOver(t, "https://a.example/1")

	if page := ranking.PageFrom(10, 10); len(page.Items) != 0 {
		t.Fatalf("PageFrom(10, 10) = %+v, want an empty page", page)
	}
}

func TestAPageOfNoItemsCarriesNothing(t *testing.T) {
	t.Parallel()

	ranking := rankingOver(t, "https://a.example/1")

	if page := ranking.PageFrom(0, 0); len(page.Items) != 0 {
		t.Fatalf("PageFrom(0, 0) = %+v, want an empty page", page)
	}
}
