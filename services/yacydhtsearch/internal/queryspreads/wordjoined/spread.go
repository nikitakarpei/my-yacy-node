// Package wordjoined finds documents that match the whole query when no single
// peer holds every query word, by joining what the peers of each word hold.
package wordjoined

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/matchingwords"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordholdings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordroles"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type RememberedDocumentAmounts interface {
	Remember(ctx context.Context, documentAmounts map[yacymodel.Hash]int)
}

type Spread struct {
	wordPartitionAsks        wordasks.WordPartitionAsks
	queryWordDocumentAmounts RememberedDocumentAmounts
	leadingWord              leadingword.Finder
	matchingWordAsker        matchingwords.Asker
	urlMetadataAsker         urlmetadataasks.Asker
	partitions               yacymodel.DHTRingPartitions
	observer                 WordJoinedSpreadObserver
}

//nolint:revive // argument-limit: the spread takes its word partition asks, word amounts, leading word, matching word asker, URL metadata asker, ring and observer
func New(
	wordPartitionAsks wordasks.WordPartitionAsks,
	queryWordDocumentAmounts RememberedDocumentAmounts,
	leadingWord leadingword.Finder,
	matchingWordAsker matchingwords.Asker,
	urlMetadataAsker urlmetadataasks.Asker,
	partitions yacymodel.DHTRingPartitions,
	observer WordJoinedSpreadObserver,
) Spread {
	return Spread{
		wordPartitionAsks:        wordPartitionAsks,
		queryWordDocumentAmounts: queryWordDocumentAmounts,
		leadingWord:              leadingWord,
		matchingWordAsker:        matchingWordAsker,
		urlMetadataAsker:         urlMetadataAsker,
		partitions:               partitions,
		observer:                 observer,
	}
}

func (spread Spread) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) queryfindings.Findings {
	startedAt := time.Now()

	wordAsksContext, endWordAsks := withinHalfOfTheTimeLeft(ctx)
	defer endWordAsks()
	run := wordasks.Start(
		wordAsksContext,
		spread.wordPartitionAsks,
		query,
		chosenPeersPerQueryWord,
		spread.partitions,
	)
	lead := spread.leadingWord.FindFor(ctx, query, run)
	roles := wordroles.From(lead, query)
	run.AskEveryPartitionFor(roles.ListingWords)
	matchingWordAskKinds := spread.matchingWordAsker.AskInEachPartition(
		roles, lead.AmountOfDocumentsInAPartition, run,
	)
	wordAnswers := run.Finish()

	holdings := wordholdings.OfEachQueryWord(query, wordAnswers, spread.partitions)
	spread.queryWordDocumentAmounts.Remember(
		ctx, holdings.AmountOfDocumentsInAPartitionPerQueryWord(),
	)
	joinedDocuments := holdings.DocumentsWithEveryWord()
	documentsWithoutMetadata := wordAnswers.DocumentsWithoutMetadataAmong(joinedDocuments)
	holders := wordAnswers.DocumentHolders()
	urlMetadata := spread.urlMetadataAsker.AskFor(ctx, documentsWithoutMetadata, holders)

	spread.observer.WordJoinedSpreadPerformed(ctx, performedWordJoinedSpreadFrom(
		wordAnswers,
		holdings,
		lead,
		matchingWordAskKinds,
		joinedDocuments,
		documentsWithoutMetadata,
		urlMetadata,
		time.Since(startedAt),
	))

	return findingsFrom(query, wordAnswers, holdings, joinedDocuments, urlMetadata)
}

func withinHalfOfTheTimeLeft(ctx context.Context) (context.Context, context.CancelFunc) {
	deadline, bounded := ctx.Deadline()
	if !bounded {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, time.Until(deadline)/2)
}
