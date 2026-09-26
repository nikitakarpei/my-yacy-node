package memory_test

import (
	"testing"
	"time"

	cachedrankingsmemory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/cachedrankings/memory"
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

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return searchresult.Ranking{
		Items: []searchresult.Item{
			{Hash: hash, Address: address},
		},
	}
}

func TestARankingIsReadBackForTheQueryItWasCachedFor(t *testing.T) {
	t.Parallel()

	cache := cachedrankingsmemory.New(capacity, lifetime)
	query := searchquery.Query{Words: []string{"berlin"}}
	cache.Store(t.Context(), query, rankingOver(t, "https://a.example/"))

	ranking, found := cache.RankingFor(t.Context(), query)

	if !found || len(ranking.Items) != 1 || ranking.Items[0].Address != "https://a.example/" {
		t.Fatalf("RankingFor = %+v (found %t), want the ranking cache", ranking, found)
	}
}

func TestNoRankingIsCachedForAQueryNobodyAsked(t *testing.T) {
	t.Parallel()

	cache := cachedrankingsmemory.New(capacity, lifetime)

	if _, found := cache.RankingFor(
		t.Context(),
		searchquery.Query{Words: []string{"berlin"}},
	); found {
		t.Fatal("RankingFor found a ranking nobody cache")
	}
}

func TestOneQueryDoesNotAnswerAnother(t *testing.T) {
	t.Parallel()

	cache := cachedrankingsmemory.New(capacity, lifetime)
	cache.Store(
		t.Context(),
		searchquery.Query{Words: []string{"berlin"}},
		rankingOver(t, "https://a.example/"),
	)

	if _, found := cache.RankingFor(
		t.Context(),
		searchquery.Query{Words: []string{"hamburg"}},
	); found {
		t.Fatal("RankingFor answered one query with another query's ranking")
	}
}

func TestTheOldestRankingGoesWhenTheCapacityIsFull(t *testing.T) {
	t.Parallel()

	cache := cachedrankingsmemory.New(capacity, lifetime)
	for _, word := range []string{"berlin", "hamburg", "bremen"} {
		cache.Store(
			t.Context(),
			searchquery.Query{Words: []string{word}},
			rankingOver(t, "https://a.example/"+word),
		)
	}

	if _, found := cache.RankingFor(
		t.Context(),
		searchquery.Query{Words: []string{"berlin"}},
	); found {
		t.Fatal("RankingFor still returns the oldest ranking past the capacity")
	}
	if _, found := cache.RankingFor(
		t.Context(),
		searchquery.Query{Words: []string{"bremen"}},
	); !found {
		t.Fatal("RankingFor dropped the newest ranking")
	}
}

func TestARankingIsGoneOnceItsLifetimeIsSpent(t *testing.T) {
	t.Parallel()

	cache := cachedrankingsmemory.New(capacity, 20*time.Millisecond)
	query := searchquery.Query{Words: []string{"berlin"}}
	cache.Store(t.Context(), query, rankingOver(t, "https://a.example/"))

	time.Sleep(200 * time.Millisecond)

	if _, found := cache.RankingFor(t.Context(), query); found {
		t.Fatal("RankingFor still returns a ranking past its lifetime")
	}
}
