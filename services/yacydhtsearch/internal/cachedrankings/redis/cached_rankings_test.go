package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	cachedrankingsredis "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/cachedrankings/redis"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/urlhashtest"
)

const lifetime = time.Hour

type recordedFailure struct {
	lookups int
	stores  int
}

func (r *recordedFailure) RankingLookupFailed(context.Context, searchquery.Query, error) {
	r.lookups++
}

func (r *recordedFailure) RankingStoreFailed(context.Context, searchquery.Query, error) {
	r.stores++
}

func rankingOver(t *testing.T, address string) searchresult.Ranking {
	t.Helper()

	hash := urlhashtest.URLHashOf(t, address)

	return searchresult.Ranking{
		Items: []searchresult.Item{{
			Hash:        hash,
			Address:     address,
			Title:       "Weather",
			Description: "prose",
		}},
	}
}

func cacheOn(
	server *miniredis.Miniredis,
	observer cachedrankingsredis.CachedRankingsObserver,
) *cachedrankingsredis.CachedRankings {
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})

	return cachedrankingsredis.New(client, lifetime, observer)
}

func TestARankingIsReadBackForTheQueryItWasCachedFor(t *testing.T) {
	t.Parallel()

	failures := &recordedFailure{}
	cache := cacheOn(miniredis.RunT(t), failures)
	query := searchquery.Query{Words: []string{"berlin"}}
	cache.Store(t.Context(), query, rankingOver(t, "https://a.example/"))

	ranking, found := cache.RankingFor(t.Context(), query)

	if !found || len(ranking.Items) != 1 {
		t.Fatalf("RankingFor = %+v (found %t), want the ranking cache", ranking, found)
	}
	item := ranking.Items[0]
	if item.Address != "https://a.example/" || item.Title != "Weather" || item.Hash.IsZero() {
		t.Fatalf("item = %+v, want every field it was cache with", item)
	}
	if failures.lookups != 0 || failures.stores != 0 {
		t.Fatalf("failures reported %+v, want none", failures)
	}
}

func TestNoRankingIsCachedForAQueryNobodyAsked(t *testing.T) {
	t.Parallel()

	failures := &recordedFailure{}
	cache := cacheOn(miniredis.RunT(t), failures)

	_, found := cache.RankingFor(t.Context(), searchquery.Query{Words: []string{"berlin"}})

	if found || failures.lookups != 0 {
		t.Fatalf(
			"RankingFor found %t with %d failures, want a plain miss",
			found,
			failures.lookups,
		)
	}
}

func TestARankingIsGoneOnceItsLifetimeIsSpent(t *testing.T) {
	t.Parallel()

	server := miniredis.RunT(t)
	cache := cacheOn(server, &recordedFailure{})
	query := searchquery.Query{Words: []string{"berlin"}}
	cache.Store(t.Context(), query, rankingOver(t, "https://a.example/"))

	server.FastForward(lifetime)

	if _, found := cache.RankingFor(t.Context(), query); found {
		t.Fatal("RankingFor still stores a ranking past its lifetime")
	}
}

func refusingServer(t *testing.T) *miniredis.Miniredis {
	t.Helper()

	server := miniredis.RunT(t)
	server.SetError("server down")

	return server
}

func TestARankingTheServerRefusesIsReported(t *testing.T) {
	t.Parallel()

	failures := &recordedFailure{}
	cache := cacheOn(refusingServer(t), failures)

	cache.Store(
		t.Context(),
		searchquery.Query{Words: []string{"berlin"}},
		rankingOver(t, "https://a.example/"),
	)

	if failures.stores != 1 {
		t.Fatalf("store failures = %d, want one", failures.stores)
	}
}

func TestALookupTheServerRefusesIsReportedAndIsAMiss(t *testing.T) {
	t.Parallel()

	failures := &recordedFailure{}
	cache := cacheOn(refusingServer(t), failures)

	_, found := cache.RankingFor(t.Context(), searchquery.Query{Words: []string{"berlin"}})

	if found || failures.lookups != 1 {
		t.Fatalf(
			"RankingFor found %t with %d failures, want a reported miss",
			found,
			failures.lookups,
		)
	}
}
