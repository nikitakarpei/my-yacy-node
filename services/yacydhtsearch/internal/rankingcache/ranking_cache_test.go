package rankingcache_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcache"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const answeredAddress = "https://a.example/"

type countedNetwork struct {
	searches int
	ranking  searchresult.Ranking
	ranked   bool
}

func (n *countedNetwork) Search(
	context.Context,
	searchquery.Query,
) (searchresult.Ranking, bool) {
	n.searches++

	return n.ranking, n.ranked
}

type rememberedRankings struct {
	cached map[string]searchresult.Ranking
}

func newRememberedRankings() *rememberedRankings {
	return &rememberedRankings{cached: map[string]searchresult.Ranking{}}
}

func (r *rememberedRankings) RankingFor(
	_ context.Context,
	query searchquery.Query,
) (searchresult.Ranking, bool) {
	ranking, cached := r.cached[query.String()]

	return ranking, cached
}

func (r *rememberedRankings) Store(
	_ context.Context,
	query searchquery.Query,
	ranking searchresult.Ranking,
) {
	r.cached[query.String()] = ranking
}

type nothingCached struct{}

func (nothingCached) RankingFor(
	context.Context,
	searchquery.Query,
) (searchresult.Ranking, bool) {
	return searchresult.Ranking{}, false
}

func (nothingCached) Store(context.Context, searchquery.Query, searchresult.Ranking) {}

type recordedReports struct {
	amountAnsweredFromCache int
	amountMissedCache       int
}

func (reports *recordedReports) QueryAnsweredFromCache(context.Context, searchquery.Query) {
	reports.amountAnsweredFromCache++
}

func (reports *recordedReports) QueryMissedCache(context.Context, searchquery.Query) {
	reports.amountMissedCache++
}

func networkAnsweringOneAddress(t *testing.T) *countedNetwork {
	t.Helper()

	return &countedNetwork{ranking: rankingOver(t, answeredAddress), ranked: true}
}

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

func TestTheNetworkAnswersAQueryNoRankingIsCachedFor(t *testing.T) {
	t.Parallel()

	network := networkAnsweringOneAddress(t)
	reports := &recordedReports{}
	cache := rankingcache.New(newRememberedRankings(), network, reports)

	ranking, ranked := cache.Search(t.Context(), searchquery.Query{Words: []string{"berlin"}})

	if !ranked || len(ranking.Items) != 1 || ranking.Items[0].Address != answeredAddress {
		t.Fatalf("Search = %+v, %t, want what the network ranked", ranking.Items, ranked)
	}
	if network.searches != 1 || reports.amountMissedCache != 1 {
		t.Fatalf(
			"network searched %d times, cache missed %d, want one of each",
			network.searches,
			reports.amountMissedCache,
		)
	}
}

func TestARepeatedQueryReachesTheNetworkOnce(t *testing.T) {
	t.Parallel()

	network := networkAnsweringOneAddress(t)
	reports := &recordedReports{}
	cache := rankingcache.New(newRememberedRankings(), network, reports)

	first, _ := cache.Search(t.Context(), searchquery.Query{Words: []string{"berlin"}})
	second, ranked := cache.Search(t.Context(), searchquery.Query{Words: []string{"berlin"}})

	if network.searches != 1 {
		t.Fatalf("network searched %d times, want once", network.searches)
	}
	if !ranked || len(second.Items) != len(first.Items) ||
		second.Items[0].Address != first.Items[0].Address {
		t.Fatalf("second = %+v, want the ranking the first query found", second.Items)
	}
	if reports.amountAnsweredFromCache != 1 || reports.amountMissedCache != 1 {
		t.Fatalf("cache reported %+v, want one hit and one miss", reports)
	}
}

func TestAnotherQueryReachesTheNetworkOfItsOwn(t *testing.T) {
	t.Parallel()

	network := networkAnsweringOneAddress(t)
	cache := rankingcache.New(newRememberedRankings(), network, &recordedReports{})

	cache.Search(t.Context(), searchquery.Query{Words: []string{"berlin"}})
	cache.Search(t.Context(), searchquery.Query{Words: []string{"hamburg"}})

	if network.searches != 2 {
		t.Fatalf("network searched %d times, want one per query", network.searches)
	}
}

func TestARankingThatIsNeverCachedSendsEveryQueryToTheNetwork(t *testing.T) {
	t.Parallel()

	network := networkAnsweringOneAddress(t)
	cache := rankingcache.New(nothingCached{}, network, &recordedReports{})

	cache.Search(t.Context(), searchquery.Query{Words: []string{"berlin"}})
	cache.Search(t.Context(), searchquery.Query{Words: []string{"berlin"}})

	if network.searches != 2 {
		t.Fatalf("network searched %d times, want one per query", network.searches)
	}
}

func TestAQueryTheNetworkMadeNoRankingForLeavesTheCacheEmpty(t *testing.T) {
	t.Parallel()

	network := &countedNetwork{ranked: false}
	reports := &recordedReports{}
	cache := rankingcache.New(newRememberedRankings(), network, reports)

	cache.Search(t.Context(), searchquery.Query{Words: []string{"berlin"}})
	second, ranked := cache.Search(t.Context(), searchquery.Query{Words: []string{"berlin"}})

	if ranked || len(second.Items) != 0 {
		t.Fatalf("second = %+v, %t, want no ranking", second.Items, ranked)
	}
	if network.searches != 2 || reports.amountAnsweredFromCache != 0 {
		t.Fatalf(
			"network searched %d times and the cache answered %d, want two and none",
			network.searches,
			reports.amountAnsweredFromCache,
		)
	}
}

func TestAnEmptyRankingTheNetworkMadeIsCachedForTheNextQuery(t *testing.T) {
	t.Parallel()

	network := &countedNetwork{ranked: true}
	reports := &recordedReports{}
	cache := rankingcache.New(newRememberedRankings(), network, reports)

	cache.Search(t.Context(), searchquery.Query{Words: []string{"berlin"}})
	cache.Search(t.Context(), searchquery.Query{Words: []string{"berlin"}})

	if network.searches != 1 || reports.amountAnsweredFromCache != 1 {
		t.Fatalf(
			"network searched %d times and the cache answered %d, want one of each",
			network.searches,
			reports.amountAnsweredFromCache,
		)
	}
}

func TestEveryObserverHearsWhetherTheCacheAnswered(t *testing.T) {
	t.Parallel()

	first, second := &recordedReports{}, &recordedReports{}
	cache := rankingcache.New(
		newRememberedRankings(),
		networkAnsweringOneAddress(t),
		rankingcache.RankingCacheObservers{first, second},
	)

	cache.Search(t.Context(), searchquery.Query{Words: []string{"berlin"}})
	cache.Search(t.Context(), searchquery.Query{Words: []string{"berlin"}})

	if first.amountMissedCache != 1 || second.amountMissedCache != 1 ||
		first.amountAnsweredFromCache != 1 || second.amountAnsweredFromCache != 1 {
		t.Fatalf("observers heard %+v and %+v, want one hit and one miss each", first, second)
	}
}
