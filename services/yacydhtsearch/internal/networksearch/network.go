// Package networksearch ranks what the peers of the network hold for one query,
// inside one whole-query time budget. It spreads the query over the peers the
// directory can ask, with the query spread it holds, puts what the peers
// answered in the order the items ordering it holds gives them, reads the pages
// of the items that came first, gives the answers what those pages say, puts
// them in order again, and carries back the items up to the ceiling as the
// ranking the client reads.
package networksearch

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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

type PageReading interface {
	PageTextPerDocument(
		ctx context.Context,
		queryWords []yacymodel.Hash,
		pagesToRead []pagereading.PageToRead,
	) map[yacymodel.URLHash]pagereading.PageText
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
	pageReading        PageReading
	itemsOrdering      ItemsOrdering
	queryBudget        time.Duration
	pagesReadPerQuery  int
	rankedItemsCeiling int
	observer           NetworkSearchObserver
}

//nolint:revive // argument-limit: what one network search holds for every query
func New(
	peerDirectory *peerdirectory.Directory,
	querySpread QuerySpread,
	pageReading PageReading,
	itemsOrdering ItemsOrdering,
	queryBudget time.Duration,
	pagesReadPerQuery int,
	rankedItemsCeiling int,
	observer NetworkSearchObserver,
) Network {
	return Network{
		peerDirectory:      peerDirectory,
		querySpread:        querySpread,
		pageReading:        pageReading,
		itemsOrdering:      itemsOrdering,
		queryBudget:        queryBudget,
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

	answers := n.querySpread.SpreadOverPeers(ctx, query, askablePeers)
	candidates := itemsUpTo(n.itemsOrdering.OrderedItemsOf(answers), n.pagesReadPerQuery)
	pageTextPerDocument := n.pageReading.PageTextPerDocument(
		ctx, query.TermHashes(), pagesToReadOf(candidates),
	)
	readAnswers := answers.CarryingTheTextOfEachDocument(
		documentTextPerDocumentOf(pageTextPerDocument),
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

func itemsUpTo(
	orderedItems []peeranswers.AnsweredItem,
	ceiling int,
) []peeranswers.AnsweredItem {
	if ceiling <= 0 {
		return nil
	}

	return orderedItems[:min(ceiling, len(orderedItems))]
}

func pagesToReadOf(candidates []peeranswers.AnsweredItem) []pagereading.PageToRead {
	pagesToRead := make([]pagereading.PageToRead, 0, len(candidates))
	for _, candidate := range candidates {
		pagesToRead = append(pagesToRead, pagereading.PageToRead{
			Document: candidate.Metadata.Hash,
			Address:  candidate.Metadata.Address,
		})
	}

	return pagesToRead
}

func documentTextPerDocumentOf(
	pageTextPerDocument map[yacymodel.URLHash]pagereading.PageText,
) map[yacymodel.URLHash]peeranswers.DocumentText {
	documentTextPerDocument := make(
		map[yacymodel.URLHash]peeranswers.DocumentText, len(pageTextPerDocument),
	)
	for document, pageText := range pageTextPerDocument {
		documentTextPerDocument[document] = peeranswers.DocumentText{
			HitsPerQueryWord: pageText.HitsPerQueryWord,
			AmountOfWords:    pageText.AmountOfWords,
			Snippet:          pageText.Snippet,
		}
	}

	return documentTextPerDocument
}

func rankingOf(rankedItems []peeranswers.AnsweredItem) searchresult.Ranking {
	items := make([]searchresult.Item, 0, len(rankedItems))
	for _, rankedItem := range rankedItems {
		items = append(items, searchresult.ItemFrom(rankedItem.Metadata))
	}

	return searchresult.Ranking{Items: items}
}
