// Package networksearch ranks what the peers of the network hold for one query,
// inside one whole-query time budget. It chooses the peers of each query word
// among those the directory can ask, lets them rest, spreads the query over them
// with the query spread it holds, puts what the peers answered in the order the
// items ordering it holds gives them, reads the pages of the items that came
// first, gives the answers what those pages say, puts them in order again, and
// carries back the items up to the ceiling as the ranking the client reads.
package networksearch

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerChoice interface {
	ChosenPeersPerQueryWordFor(
		ctx context.Context,
		queryWords []yacymodel.Hash,
		askablePeers []peerdirectory.AskablePeer,
	) peerchoice.ChosenPeersPerQueryWord
}

type QuerySpread interface {
	SpreadOverPeers(
		ctx context.Context,
		query searchquery.Query,
		chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
	) queryanswers.AnsweredQuery
}

type ItemsOrdering interface {
	OrderedItemsOf(answers queryanswers.AnsweredQuery) []queryanswers.AnsweredItem
}

type PageReading interface {
	DocumentTextPerDocument(
		ctx context.Context,
		queryWords []yacymodel.Hash,
		pagesToRead []pagereading.PageToRead,
	) map[yacymodel.URLHash]documenttext.DocumentText
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
	peerChoice         PeerChoice
	querySpread        QuerySpread
	pageReading        PageReading
	itemsOrdering      ItemsOrdering
	queryBudget        time.Duration
	pageReadBudget     time.Duration
	pagesReadPerQuery  int
	rankedItemsCeiling int
	observer           NetworkSearchObserver
}

//nolint:revive // argument-limit: what one network search holds for every query
func New(
	peerDirectory *peerdirectory.Directory,
	peerChoice PeerChoice,
	querySpread QuerySpread,
	pageReading PageReading,
	itemsOrdering ItemsOrdering,
	queryBudget time.Duration,
	pageReadBudget time.Duration,
	pagesReadPerQuery int,
	rankedItemsCeiling int,
	observer NetworkSearchObserver,
) Network {
	return Network{
		peerDirectory:      peerDirectory,
		peerChoice:         peerChoice,
		querySpread:        querySpread,
		pageReading:        pageReading,
		itemsOrdering:      itemsOrdering,
		queryBudget:        queryBudget,
		pageReadBudget:     pageReadBudget,
		pagesReadPerQuery:  pagesReadPerQuery,
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

	chosenPeersPerQueryWord := n.peerChoice.ChosenPeersPerQueryWordFor(
		ctx, query.TermHashes(), askablePeers,
	)
	n.peerDirectory.MarkPeersChosen(ctx, chosenPeersPerQueryWord.PeersAcrossQueryWords())
	spreading, endSpreading := contextOfTheQuerySpread(ctx, n.queryBudget, n.pageReadBudget)
	defer endSpreading()
	answers := n.querySpread.SpreadOverPeers(spreading, query, chosenPeersPerQueryWord)
	candidates := itemsUpTo(n.itemsOrdering.OrderedItemsOf(answers), n.pagesReadPerQuery)
	readAnswers := answers.CarryingTheTextOfEachDocument(
		n.pageReading.DocumentTextPerDocument(
			ctx, query.TermHashes(), pagesToReadOf(candidates),
		),
	)
	rankedItems := itemsUpTo(
		n.itemsOrdering.OrderedItemsOf(readAnswers), n.rankedItemsCeiling,
	)
	n.observer.NetworkSearchPerformed(
		ctx,
		performedNetworkSearchFrom(
			readAnswers, rankedItems, len(askablePeers), time.Since(startedAt),
		),
	)

	return rankingOf(rankedItems), PeersAsked
}

func contextOfTheQuerySpread(
	ctx context.Context,
	queryBudget time.Duration,
	pageReadBudget time.Duration,
) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, querySpreadBudgetFrom(queryBudget, pageReadBudget))
}

func querySpreadBudgetFrom(queryBudget time.Duration, pageReadBudget time.Duration) time.Duration {
	return max(queryBudget-pageReadBudget, 0)
}

func itemsUpTo(
	orderedItems []queryanswers.AnsweredItem,
	ceiling int,
) []queryanswers.AnsweredItem {
	if ceiling <= 0 {
		return nil
	}

	return orderedItems[:min(ceiling, len(orderedItems))]
}

func pagesToReadOf(candidates []queryanswers.AnsweredItem) []pagereading.PageToRead {
	pagesToRead := make([]pagereading.PageToRead, 0, len(candidates))
	for _, candidate := range candidates {
		pagesToRead = append(pagesToRead, pagereading.PageToRead{
			Document: candidate.Metadata.Hash,
			Address:  candidate.Metadata.Address,
		})
	}

	return pagesToRead
}

func rankingOf(rankedItems []queryanswers.AnsweredItem) searchresult.Ranking {
	items := make([]searchresult.Item, 0, len(rankedItems))
	for _, rankedItem := range rankedItems {
		items = append(items, searchresult.ItemFrom(rankedItem.Metadata))
	}

	return searchresult.Ranking{Items: items}
}
