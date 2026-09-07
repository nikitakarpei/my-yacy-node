package queryrankings_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryrankings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type countedNetwork struct {
	searches int
	ranking  searchresult.Ranking
}

func (n *countedNetwork) Search(context.Context, searchquery.Query) searchresult.Ranking {
	n.searches++

	return n.ranking
}

type rememberedRankings struct {
	cached map[string]searchresult.Ranking
}

func newRememberedRankings() *rememberedRankings {
	return &rememberedRankings{cached: map[string]searchresult.Ranking{}}
}

func (r *rememberedRankings) CachedRankingFor(
	_ context.Context,
	query searchquery.Query,
) (searchresult.Ranking, bool) {
	ranking, cached := r.cached[query.String()]

	return ranking, cached
}

func (r *rememberedRankings) StoreRanking(
	_ context.Context,
	query searchquery.Query,
	ranking searchresult.Ranking,
) {
	r.cached[query.String()] = ranking
}

type cacheNothing struct{}

func (cacheNothing) CachedRankingFor(
	context.Context,
	searchquery.Query,
) (searchresult.Ranking, bool) {
	return searchresult.Ranking{}, false
}

func (cacheNothing) StoreRanking(context.Context, searchquery.Query, searchresult.Ranking) {}

type recordedRanking struct {
	reused int
	sought int
	items  int
}

func (r *recordedRanking) CachedRankingAnswered(_ context.Context, _ searchquery.Query, items int) {
	r.reused++
	r.items = items
}

func (r *recordedRanking) NetworkRankingAnswered(
	_ context.Context,
	_ searchquery.Query,
	items int,
) {
	r.sought++
	r.items = items
}

func rankingOver(t *testing.T, address string) searchresult.Ranking {
	t.Helper()

	item, ok := searchresult.ItemFrom(yacymodel.URLMetadata{Address: address})
	if !ok {
		t.Fatalf("ItemFrom(%q) refused a well-formed address", address)
	}

	return searchresult.Ranking{Items: []searchresult.Item{item}}
}

func TestTheNetworkAnswersAQueryNoRankingIsHeldFor(t *testing.T) {
	t.Parallel()

	network := &countedNetwork{ranking: rankingOver(t, "https://a.example/")}
	observer := &recordedRanking{}
	rankings := queryrankings.New(newRememberedRankings(), network, observer)

	ranking := rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))

	if len(ranking.Items) != 1 || ranking.Items[0].Address != "https://a.example/" {
		t.Fatalf("RankingFor = %+v, want what the network found", ranking.Items)
	}
	if network.searches != 1 || observer.sought != 1 || observer.items != 1 {
		t.Fatalf("network searched %d times, sought %d", network.searches, observer.sought)
	}
}

func TestARepeatedQueryReachesTheNetworkOnce(t *testing.T) {
	t.Parallel()

	network := &countedNetwork{ranking: rankingOver(t, "https://a.example/")}
	observer := &recordedRanking{}
	rankings := queryrankings.New(newRememberedRankings(), network, observer)

	first := rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))
	second := rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))

	if network.searches != 1 {
		t.Fatalf("network searched %d times, want once", network.searches)
	}
	if len(second.Items) != len(first.Items) || second.Items[0] != first.Items[0] {
		t.Fatalf("second = %+v, want the ranking the first query found", second.Items)
	}
	if observer.reused != 1 {
		t.Fatalf("reuse reported %d times, want once", observer.reused)
	}
}

func TestAnotherQueryReachesTheNetworkOfItsOwn(t *testing.T) {
	t.Parallel()

	network := &countedNetwork{}
	rankings := queryrankings.New(newRememberedRankings(), network, &recordedRanking{})

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))
	rankings.RankingFor(t.Context(), searchquery.QueryFrom("hamburg"))

	if network.searches != 2 {
		t.Fatalf("network searched %d times, want one per query", network.searches)
	}
}

func TestARankingThatIsNeverHeldSendsEveryQueryToTheNetwork(t *testing.T) {
	t.Parallel()

	network := &countedNetwork{}
	rankings := queryrankings.New(cacheNothing{}, network, &recordedRanking{})

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))
	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))

	if network.searches != 2 {
		t.Fatalf("network searched %d times, want one per query", network.searches)
	}
}

func TestEveryObserverHearsAboutOneRanking(t *testing.T) {
	t.Parallel()

	first, second := &recordedRanking{}, &recordedRanking{}
	rankings := queryrankings.New(
		newRememberedRankings(),
		&countedNetwork{},
		queryrankings.RankingObservers{first, second},
	)

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))
	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))

	if first.sought != 1 || second.sought != 1 || first.reused != 1 || second.reused != 1 {
		t.Fatalf("observers heard %+v and %+v, want one of each", first, second)
	}
}
