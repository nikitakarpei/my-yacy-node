// Package networksearch ranks what the peers of the network hold for one query,
// within one query budget. It asks the peers that hold the words of the query,
// puts what they answered in order, reads the pages of the documents it puts first,
// and carries back the documents up to the ceiling as the ranking the client reads.
package networksearch

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
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
	) queryfindings.Findings
}

type DocumentsOrdering interface {
	OrderedDocumentsOf(findings queryfindings.Findings) []queryfindings.FoundDocument
}

type PageReading interface {
	Start(queryWords []yacymodel.Hash) PageReadingRun
}

type PageReadingRun interface {
	StartReading(ctx context.Context, pagesToRead []pagereading.PageToRead)
	PagesReadAmong(ctx context.Context, pagesWanted []pagereading.PageToRead) pagereading.PagesRead
	Finish(ctx context.Context)
}

type NetworkSearchObserver interface {
	NetworkSearchPerformed(ctx context.Context, search PerformedNetworkSearch)
	QueryReachedNoPeer(ctx context.Context, query searchquery.Query)
}

type Network struct {
	peerDirectory        *peerdirectory.Directory
	peerChoice           PeerChoice
	querySpread          QuerySpread
	pageReading          PageReading
	documentsOrdering    DocumentsOrdering
	queryBudget          time.Duration
	pageReadBudget       time.Duration
	pagesReadPerQuery    int
	pagesReadPerSite     int
	rankedItemsCeiling   int
	compoundWordsCeiling int
	observer             NetworkSearchObserver
}

//nolint:revive // argument-limit: what one network search holds for every query
func New(
	peerDirectory *peerdirectory.Directory,
	peerChoice PeerChoice,
	querySpread QuerySpread,
	pageReading PageReading,
	documentsOrdering DocumentsOrdering,
	queryBudget time.Duration,
	pageReadBudget time.Duration,
	pagesReadPerQuery int,
	pagesReadPerSite int,
	rankedItemsCeiling int,
	compoundWordsCeiling int,
	observer NetworkSearchObserver,
) Network {
	return Network{
		peerDirectory:        peerDirectory,
		peerChoice:           peerChoice,
		querySpread:          querySpread,
		pageReading:          pageReading,
		documentsOrdering:    documentsOrdering,
		queryBudget:          queryBudget,
		pageReadBudget:       pageReadBudget,
		pagesReadPerQuery:    pagesReadPerQuery,
		pagesReadPerSite:     pagesReadPerSite,
		rankedItemsCeiling:   rankedItemsCeiling,
		compoundWordsCeiling: compoundWordsCeiling,
		observer:             observer,
	}
}

func (n Network) Search(
	ctx context.Context,
	query searchquery.Query,
) (searchresult.Ranking, bool) {
	ctx, stopQueryBudget := context.WithTimeout(ctx, n.queryBudget)
	defer stopQueryBudget()
	startedAt := time.Now()

	askablePeers := n.peerDirectory.AskablePeers(ctx)
	if len(askablePeers) == 0 {
		n.observer.QueryReachedNoPeer(ctx, query)

		return searchresult.Ranking{}, false
	}

	chosenPeersPerQueryWord := n.peerChoice.ChosenPeersPerQueryWordFor(
		ctx, query.HashesOfWordsAndCompoundWordsUpTo(n.compoundWordsCeiling), askablePeers,
	)
	querySpreadContext, endTheQuerySpread := contextWithinTheQuerySpreadBudget(
		ctx, n.queryBudget, n.pageReadBudget,
	)
	defer endTheQuerySpread()
	findings := n.querySpread.SpreadOverPeers(querySpreadContext, query, chosenPeersPerQueryWord)
	documentsToRead := documentsToReadAmong(
		n.documentsOrdering.OrderedDocumentsOf(findings),
		n.pagesReadPerQuery,
		n.pagesReadPerSite,
	)
	pagesWanted := pagesToReadOf(documentsToRead)
	pageReadingRun := n.pageReading.Start(query.WordHashes())
	pageReadingRun.StartReading(ctx, pagesWanted)
	pagesRead := pageReadingRun.PagesReadAmong(ctx, pagesWanted)
	pageReadingRun.Finish(ctx)
	findingsWithReadPages := findings.
		WithReadPages(pagesRead.PageContentsPerDocument).
		WithSpamVerdicts(pagesRead.SpamVerdictPerDocument).
		WithoutDocuments(pagesRead.WithdrawnDocuments)
	rankedDocuments := documentsUpTo(
		n.documentsOrdering.OrderedDocumentsOf(findingsWithReadPages),
		n.rankedItemsCeiling,
	)
	n.observer.NetworkSearchPerformed(
		ctx,
		performedNetworkSearchFrom(
			findingsWithReadPages,
			rankedDocuments,
			len(askablePeers),
			time.Since(startedAt),
		),
	)

	return rankingOf(rankedDocuments), true
}

func contextWithinTheQuerySpreadBudget(
	ctx context.Context,
	queryBudget time.Duration,
	pageReadBudget time.Duration,
) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, max(queryBudget-pageReadBudget, 0))
}

func documentsUpTo(
	orderedDocuments []queryfindings.FoundDocument,
	ceiling int,
) []queryfindings.FoundDocument {
	if ceiling <= 0 {
		return nil
	}

	return orderedDocuments[:min(ceiling, len(orderedDocuments))]
}

func pagesToReadOf(foundDocuments []queryfindings.FoundDocument) []pagereading.PageToRead {
	pagesToRead := make([]pagereading.PageToRead, 0, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		pagesToRead = append(pagesToRead, pagereading.PageToRead{
			Document: foundDocument.Hash,
			Address:  foundDocument.Address,
		})
	}

	return pagesToRead
}

func rankingOf(rankedDocuments []queryfindings.FoundDocument) searchresult.Ranking {
	items := make([]searchresult.Item, 0, len(rankedDocuments))
	for _, rankedDocument := range rankedDocuments {
		items = append(items, searchresult.Item{
			Hash:         rankedDocument.Hash,
			Address:      rankedDocument.Address,
			Title:        rankedDocument.Title,
			Description:  rankedDocument.Snippet,
			PublishedAt:  rankedDocument.PublishedAt,
			ImageAddress: rankedDocument.FaviconAddress,
			IsSpam:       rankedDocument.IsSpam(),
		})
	}

	return searchresult.Ranking{Items: items}
}
