// Package networksearch ranks what the peers of the network hold for one query,
// inside one whole-query time budget. It spreads the query over the peers the
// directory can ask, with the query spread it holds, and ranks what the peers
// answered.
package networksearch

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
)

type QuerySpread interface {
	SpreadOverPeers(
		ctx context.Context,
		query searchquery.Query,
		askablePeers []peerdirectory.AskablePeer,
	) [][]searchresult.Item
}

type SearchOutcome int

const (
	PeersAsked SearchOutcome = iota
	NoIndexedTermInQuery
	NoPeerToAsk
)

type NetworkSearchObserver interface {
	NetworkSearchPerformed(ctx context.Context, search PerformedNetworkSearch)
}

type Network struct {
	peerDirectory      *peerdirectory.Directory
	querySpread        QuerySpread
	queryBudget        time.Duration
	rankedItemsCeiling int
	observer           NetworkSearchObserver
}

func New(
	peerDirectory *peerdirectory.Directory,
	querySpread QuerySpread,
	queryBudget time.Duration,
	rankedItemsCeiling int,
	observer NetworkSearchObserver,
) Network {
	return Network{
		peerDirectory:      peerDirectory,
		querySpread:        querySpread,
		queryBudget:        queryBudget,
		rankedItemsCeiling: rankedItemsCeiling,
		observer:           observer,
	}
}

func (n Network) Search(
	ctx context.Context,
	query searchquery.Query,
) (searchresult.Ranking, SearchOutcome) {
	if len(query.Terms) == 0 {
		return searchresult.Ranking{}, NoIndexedTermInQuery
	}

	ctx, stopQueryBudget := context.WithTimeout(ctx, n.queryBudget)
	defer stopQueryBudget()
	startedAt := time.Now()

	askablePeers := n.peerDirectory.AskablePeers(ctx)
	if len(askablePeers) == 0 {
		return searchresult.Ranking{}, NoPeerToAsk
	}

	itemsOfEachPeer := n.querySpread.SpreadOverPeers(ctx, query, askablePeers)
	ranking := searchresult.RankingFrom(itemsOfEachPeer, n.rankedItemsCeiling)
	n.observer.NetworkSearchPerformed(
		ctx,
		performedNetworkSearchFrom(
			itemsOfEachPeer, ranking, len(askablePeers), time.Since(startedAt),
		),
	)

	return ranking, PeersAsked
}
