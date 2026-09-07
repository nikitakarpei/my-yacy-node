package memory_test

import (
	"testing"
	"time"

	rankingcachememory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcache/memory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	capacity = 2
	lifetime = time.Minute
)

func rankingOver(t *testing.T, address string) searchresult.Ranking {
	t.Helper()

	item, ok := searchresult.ItemFrom(yacymodel.URLMetadata{Address: address})
	if !ok {
		t.Fatalf("ItemFrom(%q) refused a well-formed address", address)
	}

	return searchresult.Ranking{Items: []searchresult.Item{item}}
}

func TestARankingIsReadBackForTheQueryItWasHeldFor(t *testing.T) {
	t.Parallel()

	cache := rankingcachememory.New(capacity, lifetime)
	query := searchquery.QueryFrom("berlin")
	cache.StoreRanking(t.Context(), query, rankingOver(t, "https://a.example/"))

	ranking, found := cache.CachedRankingFor(t.Context(), query)

	if !found || len(ranking.Items) != 1 || ranking.Items[0].Address != "https://a.example/" {
		t.Fatalf("CachedRankingFor = %+v (found %t), want the ranking cache", ranking, found)
	}
}

func TestNoRankingIsHeldForAQueryNobodyAsked(t *testing.T) {
	t.Parallel()

	cache := rankingcachememory.New(capacity, lifetime)

	if _, found := cache.CachedRankingFor(t.Context(), searchquery.QueryFrom("berlin")); found {
		t.Fatal("CachedRankingFor found a ranking nobody cache")
	}
}

func TestOneQueryDoesNotAnswerAnother(t *testing.T) {
	t.Parallel()

	cache := rankingcachememory.New(capacity, lifetime)
	cache.StoreRanking(
		t.Context(), searchquery.QueryFrom("berlin"), rankingOver(t, "https://a.example/"),
	)

	if _, found := cache.CachedRankingFor(t.Context(), searchquery.QueryFrom("hamburg")); found {
		t.Fatal("CachedRankingFor answered one query with another query's ranking")
	}
}

func TestTheOldestRankingGoesWhenTheCapacityIsFull(t *testing.T) {
	t.Parallel()

	cache := rankingcachememory.New(capacity, lifetime)
	for _, term := range []string{"berlin", "hamburg", "bremen"} {
		cache.StoreRanking(
			t.Context(), searchquery.QueryFrom(term), rankingOver(t, "https://a.example/"+term),
		)
	}

	if _, found := cache.CachedRankingFor(t.Context(), searchquery.QueryFrom("berlin")); found {
		t.Fatal("CachedRankingFor still returns the oldest ranking past the capacity")
	}
	if _, found := cache.CachedRankingFor(t.Context(), searchquery.QueryFrom("bremen")); !found {
		t.Fatal("CachedRankingFor dropped the newest ranking")
	}
}

func TestARankingIsGoneOnceItsLifetimeIsSpent(t *testing.T) {
	t.Parallel()

	cache := rankingcachememory.New(capacity, 20*time.Millisecond)
	query := searchquery.QueryFrom("berlin")
	cache.StoreRanking(t.Context(), query, rankingOver(t, "https://a.example/"))

	time.Sleep(200 * time.Millisecond)

	if _, found := cache.CachedRankingFor(t.Context(), query); found {
		t.Fatal("CachedRankingFor still returns a ranking past its lifetime")
	}
}
