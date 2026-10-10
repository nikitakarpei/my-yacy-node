// Package networksearch ranks what the peers of the network hold for one query,
// within one query budget. It asks the peers that hold the words of the query,
// puts what they answered in order, reads the pages of the documents it puts first,
// can leave out each document whose page it did not read, and carries back the
// documents up to the ceiling as the ranking the client reads.
package networksearch

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
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
		growth queryfindings.Growth,
	) queryfindings.Findings
}

type DocumentsOrdering interface {
	OrderedDocumentsOf(findings queryfindings.Findings) []queryfindings.FoundDocument
}

type PageReading interface {
	Start(queryWords []string) PageReadingRun
}

type PageReadingRun interface {
	ReadAhead(ctx context.Context, pagesToRead []pagereading.PageToRead)
	Abandon(pagesToAbandon []pagereading.PageToRead)
	Read(ctx context.Context, pagesWanted []pagereading.PageToRead) pagereading.PagesRead
	Finish(ctx context.Context)
}

type NetworkSearchObserver interface {
	NetworkSearchPerformed(ctx context.Context, search PerformedNetworkSearch)
	QueryReachedNoPeer(ctx context.Context, query searchquery.Query)
}

type Network struct {
	peerDirectory            *peerdirectory.Directory
	peerChoice               PeerChoice
	querySpread              QuerySpread
	pageReading              PageReading
	documentsOrdering        DocumentsOrdering
	hideUnreadResults        bool
	queryBudget              time.Duration
	pagesReadPerQueryCeiling int
	pagesReadPerSiteCeiling  int
	rankedItemsCeiling       int
	compoundWordsCeiling     int
	observer                 NetworkSearchObserver
}

//nolint:revive // argument-limit: what one network search holds for every query
func New(
	peerDirectory *peerdirectory.Directory,
	peerChoice PeerChoice,
	querySpread QuerySpread,
	pageReading PageReading,
	documentsOrdering DocumentsOrdering,
	hideUnreadResults bool,
	queryBudget time.Duration,
	pagesReadPerQueryCeiling int,
	pagesReadPerSiteCeiling int,
	rankedItemsCeiling int,
	compoundWordsCeiling int,
	observer NetworkSearchObserver,
) Network {
	return Network{
		peerDirectory:            peerDirectory,
		peerChoice:               peerChoice,
		querySpread:              querySpread,
		pageReading:              pageReading,
		documentsOrdering:        documentsOrdering,
		hideUnreadResults:        hideUnreadResults,
		queryBudget:              queryBudget,
		pagesReadPerQueryCeiling: pagesReadPerQueryCeiling,
		pagesReadPerSiteCeiling:  pagesReadPerSiteCeiling,
		rankedItemsCeiling:       rankedItemsCeiling,
		compoundWordsCeiling:     compoundWordsCeiling,
		observer:                 observer,
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
	pageReadingRun := n.pageReading.Start(query.Words)
	prefetcher := n.prefetcherFor(pageReadingRun)
	prefetcher.Start(ctx)
	findings := n.querySpread.SpreadOverPeers(
		ctx, query, chosenPeersPerQueryWord, prefetcher,
	)
	prefetcher.Stop()
	pagesWanted := pagesToReadAmong(
		n.documentsOrdering.OrderedDocumentsOf(findings),
		n.pagesReadPerQueryCeiling,
		n.pagesReadPerSiteCeiling,
	)
	pagesRead := pageReadingRun.Read(ctx, pagesWanted)
	pageReadingRun.Finish(ctx)
	readFindings := n.findingsWithReadPagesFrom(findings, pagesRead.PageContentsPerDocument).
		WithSpamVerdicts(pagesRead.SpamVerdictPerDocument).
		WithoutDocuments(pagesRead.WithdrawnDocuments)
	rankedDocuments := documentsUpTo(
		n.documentsOrdering.OrderedDocumentsOf(readFindings),
		n.rankedItemsCeiling,
	)
	n.observer.NetworkSearchPerformed(
		ctx,
		performedNetworkSearchFrom(
			readFindings,
			rankedDocuments,
			len(askablePeers),
			time.Since(startedAt),
		),
	)

	return rankingOf(rankedDocuments), true
}

func (n Network) findingsWithReadPagesFrom(
	findings queryfindings.Findings,
	pageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents,
) queryfindings.Findings {
	if n.hideUnreadResults {
		return findings.WithOnlyReadPages(pageContentsPerDocument)
	}

	return findings.WithReadPages(pageContentsPerDocument)
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
