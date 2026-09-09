package queryrankings_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryrankings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const answeredAddress = "https://a.example/"

type countedNetwork struct {
	searches int
	ranking  searchresult.Ranking
	outcome  networksearch.SearchOutcome
}

func (n *countedNetwork) Search(
	context.Context,
	searchquery.Query,
) (searchresult.Ranking, networksearch.SearchOutcome) {
	n.searches++

	return n.ranking, n.outcome
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
	fromCache          int
	fromPeers          int
	withoutIndexedTerm int
	withoutPeerToAsk   int
	items              int
}

func (r *recordedRanking) QueryAnsweredFromCache(
	_ context.Context,
	_ searchquery.Query,
	items int,
) {
	r.fromCache++
	r.items = items
}

func (r *recordedRanking) QueryAnsweredByPeers(_ context.Context, _ searchquery.Query, items int) {
	r.fromPeers++
	r.items = items
}

func (r *recordedRanking) QueryHadNoIndexedTerm(context.Context, searchquery.Query) {
	r.withoutIndexedTerm++
}

func (r *recordedRanking) QueryFoundNoPeerToAsk(context.Context, searchquery.Query) {
	r.withoutPeerToAsk++
}

func networkAnsweringOneAddress(t *testing.T) *countedNetwork {
	t.Helper()

	return &countedNetwork{
		ranking: rankingOver(t, answeredAddress),
		outcome: networksearch.PeersAsked,
	}
}

func rankingOver(t *testing.T, address string) searchresult.Ranking {
	t.Helper()

	item, ok := searchresult.ItemFrom(yacymodel.URLMetadata{Address: address})
	if !ok {
		t.Fatalf("ItemFrom(%q) refused a well-formed address", address)
	}

	return searchresult.Ranking{Items: []searchresult.Item{item}}
}

func TestThePeersAnswerAQueryNoRankingIsHeldFor(t *testing.T) {
	t.Parallel()

	network := networkAnsweringOneAddress(t)
	observer := &recordedRanking{}
	rankings := queryrankings.New(newRememberedRankings(), network, observer)

	ranking := rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))

	if len(ranking.Items) != 1 || ranking.Items[0].Address != answeredAddress {
		t.Fatalf("RankingFor = %+v, want what the peers hold", ranking.Items)
	}
	if network.searches != 1 || observer.fromPeers != 1 || observer.items != 1 {
		t.Fatalf(
			"network searched %d times, answered by peers %d",
			network.searches,
			observer.fromPeers,
		)
	}
}

func TestARepeatedQueryReachesTheNetworkOnce(t *testing.T) {
	t.Parallel()

	network := networkAnsweringOneAddress(t)
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
	if observer.fromCache != 1 {
		t.Fatalf("cache answered %d times, want once", observer.fromCache)
	}
}

func TestAnotherQueryReachesTheNetworkOfItsOwn(t *testing.T) {
	t.Parallel()

	network := networkAnsweringOneAddress(t)
	rankings := queryrankings.New(newRememberedRankings(), network, &recordedRanking{})

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))
	rankings.RankingFor(t.Context(), searchquery.QueryFrom("hamburg"))

	if network.searches != 2 {
		t.Fatalf("network searched %d times, want one per query", network.searches)
	}
}

func TestARankingThatIsNeverHeldSendsEveryQueryToTheNetwork(t *testing.T) {
	t.Parallel()

	network := networkAnsweringOneAddress(t)
	rankings := queryrankings.New(cacheNothing{}, network, &recordedRanking{})

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))
	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))

	if network.searches != 2 {
		t.Fatalf("network searched %d times, want one per query", network.searches)
	}
}

func TestAQueryWithoutAnIndexedTermIsReportedAndLeavesTheCacheEmpty(t *testing.T) {
	t.Parallel()

	network := &countedNetwork{outcome: networksearch.NoIndexedTermInQuery}
	observer := &recordedRanking{}
	rankings := queryrankings.New(newRememberedRankings(), network, observer)

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))
	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))

	if observer.withoutIndexedTerm != 2 || observer.fromCache != 0 {
		t.Fatalf(
			"reported %d queries without an indexed term and %d from the cache, want two and none",
			observer.withoutIndexedTerm,
			observer.fromCache,
		)
	}
}

func TestAQueryThatFoundNoPeerToAskIsReportedAndLeavesTheCacheEmpty(t *testing.T) {
	t.Parallel()

	network := &countedNetwork{outcome: networksearch.NoPeerToAsk}
	observer := &recordedRanking{}
	rankings := queryrankings.New(newRememberedRankings(), network, observer)

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))
	second := rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))

	if network.searches != 2 || observer.withoutPeerToAsk != 2 {
		t.Fatalf(
			"network searched %d times and reported %d queries without a peer, want two of each",
			network.searches,
			observer.withoutPeerToAsk,
		)
	}
	if len(second.Items) != 0 || observer.fromCache != 0 {
		t.Fatalf(
			"second = %+v answered from the cache %d times, want an empty ranking and none",
			second.Items,
			observer.fromCache,
		)
	}
}

func TestAnEmptyRankingThePeersAnsweredIsHeldForTheNextQuery(t *testing.T) {
	t.Parallel()

	network := &countedNetwork{outcome: networksearch.PeersAsked}
	observer := &recordedRanking{}
	rankings := queryrankings.New(newRememberedRankings(), network, observer)

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))
	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))

	if network.searches != 1 || observer.fromCache != 1 {
		t.Fatalf(
			"network searched %d times and the cache answered %d, want one of each",
			network.searches,
			observer.fromCache,
		)
	}
}

func TestEveryObserverHearsAboutOneRanking(t *testing.T) {
	t.Parallel()

	first, second := &recordedRanking{}, &recordedRanking{}
	rankings := queryrankings.New(
		newRememberedRankings(),
		networkAnsweringOneAddress(t),
		queryrankings.RankingObservers{first, second},
	)

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))
	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))

	if first.fromPeers != 1 || second.fromPeers != 1 ||
		first.fromCache != 1 || second.fromCache != 1 {
		t.Fatalf("observers heard %+v and %+v, want one of each", first, second)
	}
}

func TestEveryObserverHearsAboutAQueryThatReachedNoPeer(t *testing.T) {
	t.Parallel()

	first, second := &recordedRanking{}, &recordedRanking{}
	rankings := queryrankings.New(
		newRememberedRankings(),
		&countedNetwork{outcome: networksearch.NoPeerToAsk},
		queryrankings.RankingObservers{first, second},
	)

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))

	if first.withoutPeerToAsk != 1 || second.withoutPeerToAsk != 1 {
		t.Fatalf("observers heard %+v and %+v, want one report each", first, second)
	}
}

func TestEveryObserverHearsAboutAQueryWithoutAnIndexedTerm(t *testing.T) {
	t.Parallel()

	first, second := &recordedRanking{}, &recordedRanking{}
	rankings := queryrankings.New(
		newRememberedRankings(),
		&countedNetwork{outcome: networksearch.NoIndexedTermInQuery},
		queryrankings.RankingObservers{first, second},
	)

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin"))

	if first.withoutIndexedTerm != 1 || second.withoutIndexedTerm != 1 {
		t.Fatalf("observers heard %+v and %+v, want one report each", first, second)
	}
}
