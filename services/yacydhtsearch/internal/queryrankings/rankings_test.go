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

type cacheHoldingNothing struct{}

func (cacheHoldingNothing) CachedRankingFor(
	context.Context,
	searchquery.Query,
) (searchresult.Ranking, bool) {
	return searchresult.Ranking{}, false
}

func (cacheHoldingNothing) StoreRanking(context.Context, searchquery.Query, searchresult.Ranking) {}

type recordedReports struct {
	amountAnsweredFromCache    int
	amountAnsweredByPeers      int
	amountHoldingNoIndexedTerm int
	amountReachedNoPeer        int
	amountOfItems              int
}

func (reports *recordedReports) QueryAnsweredFromCache(
	_ context.Context,
	_ searchquery.Query,
	amountOfItems int,
) {
	reports.amountAnsweredFromCache++
	reports.amountOfItems = amountOfItems
}

func (reports *recordedReports) QueryAnsweredByPeers(
	_ context.Context,
	_ searchquery.Query,
	amountOfItems int,
) {
	reports.amountAnsweredByPeers++
	reports.amountOfItems = amountOfItems
}

func (reports *recordedReports) QueryHoldsNoIndexedTerm(context.Context, searchquery.Query) {
	reports.amountHoldingNoIndexedTerm++
}

func (reports *recordedReports) QueryReachedNoPeer(context.Context, searchquery.Query) {
	reports.amountReachedNoPeer++
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

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return searchresult.Ranking{
		Items: []searchresult.Item{
			searchresult.ItemFrom(yacymodel.URLMetadata{Hash: hash, Address: address}),
		},
	}
}

func TestThePeersAnswerAQueryNoRankingIsHeldFor(t *testing.T) {
	t.Parallel()

	network := networkAnsweringOneAddress(t)
	reports := &recordedReports{}
	rankings := queryrankings.New(newRememberedRankings(), network, reports)

	ranking := rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin", ""))

	if len(ranking.Items) != 1 || ranking.Items[0].Address != answeredAddress {
		t.Fatalf("RankingFor = %+v, want what the peers hold", ranking.Items)
	}
	if network.searches != 1 || reports.amountAnsweredByPeers != 1 || reports.amountOfItems != 1 {
		t.Fatalf(
			"network searched %d times, answered by peers %d",
			network.searches,
			reports.amountAnsweredByPeers,
		)
	}
}

func TestARepeatedQueryReachesTheNetworkOnce(t *testing.T) {
	t.Parallel()

	network := networkAnsweringOneAddress(t)
	reports := &recordedReports{}
	rankings := queryrankings.New(newRememberedRankings(), network, reports)

	first := rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin", ""))
	second := rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin", ""))

	if network.searches != 1 {
		t.Fatalf("network searched %d times, want once", network.searches)
	}
	if len(second.Items) != len(first.Items) ||
		second.Items[0].Address != first.Items[0].Address {
		t.Fatalf("second = %+v, want the ranking the first query found", second.Items)
	}
	if reports.amountAnsweredFromCache != 1 {
		t.Fatalf("cache answered %d times, want once", reports.amountAnsweredFromCache)
	}
}

func TestAnotherQueryReachesTheNetworkOfItsOwn(t *testing.T) {
	t.Parallel()

	network := networkAnsweringOneAddress(t)
	rankings := queryrankings.New(newRememberedRankings(), network, &recordedReports{})

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin", ""))
	rankings.RankingFor(t.Context(), searchquery.QueryFrom("hamburg", ""))

	if network.searches != 2 {
		t.Fatalf("network searched %d times, want one per query", network.searches)
	}
}

func TestARankingThatIsNeverHeldSendsEveryQueryToTheNetwork(t *testing.T) {
	t.Parallel()

	network := networkAnsweringOneAddress(t)
	rankings := queryrankings.New(cacheHoldingNothing{}, network, &recordedReports{})

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin", ""))
	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin", ""))

	if network.searches != 2 {
		t.Fatalf("network searched %d times, want one per query", network.searches)
	}
}

func TestAQueryWithoutAnIndexedTermIsReportedAndLeavesTheCacheEmpty(t *testing.T) {
	t.Parallel()

	network := &countedNetwork{outcome: networksearch.NoIndexedTermInQuery}
	reports := &recordedReports{}
	rankings := queryrankings.New(newRememberedRankings(), network, reports)

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin", ""))
	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin", ""))

	if reports.amountHoldingNoIndexedTerm != 2 || reports.amountAnsweredFromCache != 0 {
		t.Fatalf(
			"reported %d queries without an indexed term and %d from the cache, want two and none",
			reports.amountHoldingNoIndexedTerm,
			reports.amountAnsweredFromCache,
		)
	}
}

func TestAQueryThatReachedNoPeerIsReportedAndLeavesTheCacheEmpty(t *testing.T) {
	t.Parallel()

	network := &countedNetwork{outcome: networksearch.NoPeerToAsk}
	reports := &recordedReports{}
	rankings := queryrankings.New(newRememberedRankings(), network, reports)

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin", ""))
	second := rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin", ""))

	if network.searches != 2 || reports.amountReachedNoPeer != 2 {
		t.Fatalf(
			"network searched %d times and reported %d queries without a peer, want two of each",
			network.searches,
			reports.amountReachedNoPeer,
		)
	}
	if len(second.Items) != 0 || reports.amountAnsweredFromCache != 0 {
		t.Fatalf(
			"second = %+v answered from the cache %d times, want an empty ranking and none",
			second.Items,
			reports.amountAnsweredFromCache,
		)
	}
}

func TestAnEmptyRankingThePeersAnsweredIsHeldForTheNextQuery(t *testing.T) {
	t.Parallel()

	network := &countedNetwork{outcome: networksearch.PeersAsked}
	reports := &recordedReports{}
	rankings := queryrankings.New(newRememberedRankings(), network, reports)

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin", ""))
	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin", ""))

	if network.searches != 1 || reports.amountAnsweredFromCache != 1 {
		t.Fatalf(
			"network searched %d times and the cache answered %d, want one of each",
			network.searches,
			reports.amountAnsweredFromCache,
		)
	}
}

func TestEveryObserverHearsAboutOneRanking(t *testing.T) {
	t.Parallel()

	first, second := &recordedReports{}, &recordedReports{}
	rankings := queryrankings.New(
		newRememberedRankings(),
		networkAnsweringOneAddress(t),
		queryrankings.QueryRankingObservers{first, second},
	)

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin", ""))
	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin", ""))

	if first.amountAnsweredByPeers != 1 || second.amountAnsweredByPeers != 1 ||
		first.amountAnsweredFromCache != 1 || second.amountAnsweredFromCache != 1 {
		t.Fatalf("observers heard %+v and %+v, want one of each", first, second)
	}
}

func TestEveryObserverHearsAboutAQueryThatReachedNoPeer(t *testing.T) {
	t.Parallel()

	first, second := &recordedReports{}, &recordedReports{}
	rankings := queryrankings.New(
		newRememberedRankings(),
		&countedNetwork{outcome: networksearch.NoPeerToAsk},
		queryrankings.QueryRankingObservers{first, second},
	)

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin", ""))

	if first.amountReachedNoPeer != 1 || second.amountReachedNoPeer != 1 {
		t.Fatalf("observers heard %+v and %+v, want one report each", first, second)
	}
}

func TestEveryObserverHearsAboutAQueryWithoutAnIndexedTerm(t *testing.T) {
	t.Parallel()

	first, second := &recordedReports{}, &recordedReports{}
	rankings := queryrankings.New(
		newRememberedRankings(),
		&countedNetwork{outcome: networksearch.NoIndexedTermInQuery},
		queryrankings.QueryRankingObservers{first, second},
	)

	rankings.RankingFor(t.Context(), searchquery.QueryFrom("berlin", ""))

	if first.amountHoldingNoIndexedTerm != 1 || second.amountHoldingNoIndexedTerm != 1 {
		t.Fatalf("observers heard %+v and %+v, want one report each", first, second)
	}
}
