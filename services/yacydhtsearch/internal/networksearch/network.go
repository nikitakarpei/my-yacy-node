// Package networksearch ranks what the peers of the network hold for one query,
// inside one whole-query time budget. It spreads the query over the peers the
// directory can ask, with the query spread it holds, puts what the peers
// answered in the order the items ordering it holds gives them, and carries
// back the items up to the ceiling as the ranking the client reads.
package networksearch

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
)

type QuerySpread interface {
	SpreadOverPeers(
		ctx context.Context,
		query searchquery.Query,
		askablePeers []peerdirectory.AskablePeer,
	) peeranswers.AnsweredQuery
}

type ItemsOrdering interface {
	OrderedItemsOf(answers peeranswers.AnsweredQuery) []peeranswers.AnsweredItem
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
	itemsOrdering      ItemsOrdering
	queryBudget        time.Duration
	rankedItemsCeiling int
	observer           NetworkSearchObserver
}

//nolint:revive // argument-limit: what one network search holds for every query
func New(
	peerDirectory *peerdirectory.Directory,
	querySpread QuerySpread,
	itemsOrdering ItemsOrdering,
	queryBudget time.Duration,
	rankedItemsCeiling int,
	observer NetworkSearchObserver,
) Network {
	return Network{
		peerDirectory:      peerDirectory,
		querySpread:        querySpread,
		itemsOrdering:      itemsOrdering,
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

	answers := n.querySpread.SpreadOverPeers(ctx, query, askablePeers)
	orderedItems := n.itemsOrdering.OrderedItemsOf(answers)
	rankedItems := n.rankedItemsAmong(orderedItems)
	n.observer.NetworkSearchPerformed(
		ctx,
		performedNetworkSearchFrom(answers, rankedItems, len(askablePeers), time.Since(startedAt)),
	)

	return rankingOf(rankedItems), PeersAsked
}

func (n Network) rankedItemsAmong(
	orderedItems []peeranswers.AnsweredItem,
) []peeranswers.AnsweredItem {
	if n.rankedItemsCeiling <= 0 {
		return nil
	}

	return orderedItems[:min(n.rankedItemsCeiling, len(orderedItems))]
}

func rankingOf(rankedItems []peeranswers.AnsweredItem) searchresult.Ranking {
	items := make([]searchresult.Item, 0, len(rankedItems))
	for _, rankedItem := range rankedItems {
		items = append(items, searchresult.ItemFrom(rankedItem.Metadata))
	}

	return searchresult.Ranking{Items: items}
}
