// Package networksearch ranks what the peers of the network hold for one query,
// within one query budget. It asks the peers that hold the words of the query,
// puts what they answered in order, reads the pages of the items it puts first,
// and carries back the items up to the ceiling as the ranking the client reads.
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
	OrderedItemsOf(answers queryanswers.AnsweredQuery) []queryanswers.FoundDocument
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
	querySpreadContext, endTheQuerySpread := contextWithinTheQuerySpreadBudget(
		ctx, n.queryBudget, n.pageReadBudget,
	)
	defer endTheQuerySpread()
	answers := n.querySpread.SpreadOverPeers(querySpreadContext, query, chosenPeersPerQueryWord)
	itemsOrderedFirst := itemsUpTo(n.itemsOrdering.OrderedItemsOf(answers), n.pagesReadPerQuery)
	documentTextPerDocument := n.pageReading.DocumentTextPerDocument(
		ctx, query.TermHashes(), pagesToReadOf(itemsOrderedFirst),
	)
	answersSaturatedWithDocumentText := answers.SaturatedWith(documentTextPerDocument)
	rankedItems := itemsUpTo(
		n.itemsOrdering.OrderedItemsOf(answersSaturatedWithDocumentText), n.rankedItemsCeiling,
	)
	n.observer.NetworkSearchPerformed(
		ctx,
		performedNetworkSearchFrom(
			answersSaturatedWithDocumentText, rankedItems, len(askablePeers), time.Since(startedAt),
		),
	)

	return rankingOf(rankedItems), PeersAsked
}

func contextWithinTheQuerySpreadBudget(
	ctx context.Context,
	queryBudget time.Duration,
	pageReadBudget time.Duration,
) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, max(queryBudget-pageReadBudget, 0))
}

func itemsUpTo(
	orderedItems []queryanswers.FoundDocument,
	ceiling int,
) []queryanswers.FoundDocument {
	if ceiling <= 0 {
		return nil
	}

	return orderedItems[:min(ceiling, len(orderedItems))]
}

func pagesToReadOf(items []queryanswers.FoundDocument) []pagereading.PageToRead {
	pagesToRead := make([]pagereading.PageToRead, 0, len(items))
	for _, item := range items {
		pagesToRead = append(pagesToRead, pagereading.PageToRead{
			Document: item.Hash,
			Address:  item.Address,
		})
	}

	return pagesToRead
}

func rankingOf(rankedItems []queryanswers.FoundDocument) searchresult.Ranking {
	items := make([]searchresult.Item, 0, len(rankedItems))
	for _, rankedItem := range rankedItems {
		items = append(items, searchresult.Item{
			Hash:         rankedItem.Hash,
			Address:      rankedItem.Address,
			Title:        rankedItem.Title,
			Description:  rankedItem.Snippet,
			PublishedAt:  rankedItem.PublishedAt,
			ImageAddress: rankedItem.FaviconAddress,
		})
	}

	return searchresult.Ranking{Items: items}
}
