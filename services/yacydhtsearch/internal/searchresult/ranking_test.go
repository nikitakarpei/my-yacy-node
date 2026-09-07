package searchresult_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func itemAt(t *testing.T, address string) searchresult.Item {
	t.Helper()

	item, ok := searchresult.ItemFrom(yacymodel.URLMetadata{Address: address})
	if !ok {
		t.Fatalf("ItemFrom(%q) refused a well-formed address", address)
	}

	return item
}

func addressesOf(items []searchresult.Item) []string {
	addresses := make([]string, 0, len(items))
	for _, item := range items {
		addresses = append(addresses, item.Address)
	}

	return addresses
}

func TestRankingFromTakesOneItemPerPeerPerRound(t *testing.T) {
	t.Parallel()

	first := []searchresult.Item{itemAt(t, "https://a.example/1"), itemAt(t, "https://a.example/2")}
	second := []searchresult.Item{itemAt(t, "https://b.example/1")}

	ranking := searchresult.RankingFrom([][]searchresult.Item{first, second}, 10)

	want := []string{"https://a.example/1", "https://b.example/1", "https://a.example/2"}
	if got := addressesOf(ranking.Items); !slices.Equal(got, want) {
		t.Fatalf("RankingFrom = %v, want %v", got, want)
	}
}

func TestRankingFromReturnsOneURLOnceHoweverManyPeersHoldIt(t *testing.T) {
	t.Parallel()

	shared := itemAt(t, "https://shared.example/")
	ranking := searchresult.RankingFrom([][]searchresult.Item{{shared}, {shared}}, 10)

	if got := addressesOf(ranking.Items); !slices.Equal(got, []string{"https://shared.example/"}) {
		t.Fatalf("RankingFrom = %v, want the shared address once", got)
	}
}

func TestRankingFromStopsAtTheCeiling(t *testing.T) {
	t.Parallel()

	answer := make([]searchresult.Item, 0, 10)
	for i := range 10 {
		answer = append(answer, itemAt(t, "https://a.example/"+strconv.Itoa(i)))
	}

	ranking := searchresult.RankingFrom([][]searchresult.Item{answer}, 3)

	if len(ranking.Items) != 3 {
		t.Fatalf("RankingFrom returned %d items, want 3", len(ranking.Items))
	}
}

func TestRankingFromHoldsNothingWhenNoItemWasAskedFor(t *testing.T) {
	t.Parallel()

	answer := []searchresult.Item{itemAt(t, "https://a.example/")}

	if ranking := searchresult.RankingFrom(
		[][]searchresult.Item{answer},
		0,
	); len(
		ranking.Items,
	) != 0 {
		t.Fatalf("RankingFrom = %+v, want an empty ranking", ranking)
	}
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
