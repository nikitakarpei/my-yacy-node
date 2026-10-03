// Package wordjoined finds documents that match the whole query when no single
// peer holds every query word, by joining what the peers of each word hold.
package wordjoined

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentsperword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordroles"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type RememberedDocumentAmounts interface {
	Remember(ctx context.Context, documentAmounts map[yacymodel.Hash]int)
}

type Spread struct {
	documentsAsker       documentasks.Asker
	documentAmountsCache RememberedDocumentAmounts
	leadingWordFinder    leadingword.Finder
	urlMetadataAsker     urlmetadataasks.Asker
	observer             WordJoinedSpreadObserver
}

func New(
	documentsAsker documentasks.Asker,
	documentAmountsCache RememberedDocumentAmounts,
	leadingWordFinder leadingword.Finder,
	urlMetadataAsker urlmetadataasks.Asker,
	observer WordJoinedSpreadObserver,
) Spread {
	return Spread{
		documentsAsker:       documentsAsker,
		documentAmountsCache: documentAmountsCache,
		leadingWordFinder:    leadingWordFinder,
		urlMetadataAsker:     urlMetadataAsker,
		observer:             observer,
	}
}

func (spread Spread) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) queryfindings.Findings {
	startedAt := time.Now()

	inquiryContext, endInquiry := withinHalfOfTheTimeLeft(ctx)
	defer endInquiry()
	inquiry := spread.documentsAsker.Begin(inquiryContext, query, chosenPeersPerQueryWord)
	defer inquiry.End()

	var documentAnswers documentasks.Answers
	leadingWord := spread.leadingWordFinder.FindFor(ctx, query, inquiry)
	if lead, found := leadingWord.Get(); found {
		roles := wordroles.Around(lead, query)
		inquiry.ExpectInAPartition(lead.Word, lead.AmountOfDocumentsInAPartition)
		documentAnswers = inquiry.WhichDocumentsHavingTheseAlsoHave(
			roles.LeadAndItsCompoundWords,
			roles.OtherWords,
		)
	} else {
		documentAnswers = inquiry.WhichDocumentsHave(query.HashesOfWordsAndCompoundWords())
	}

	documentsPerWord := documentsperword.From(query, documentAnswers)
	joinedDocuments := documentsPerWord.WithEveryWord()
	documentsWithoutMetadata := documentAnswers.DocumentsWithoutMetadataAmong(joinedDocuments)
	holdersOfDocumentsWithoutMetadata := documentAnswers.HoldersOf(documentsWithoutMetadata)
	urlMetadataAnswers := spread.urlMetadataAsker.AskFor(ctx, holdersOfDocumentsWithoutMetadata)

	spread.documentAmountsCache.Remember(ctx, documentsPerWord.AmountInAPartitionPerQueryWord())
	spread.observer.WordJoinedSpreadPerformed(ctx, performedWordJoinedSpreadFrom(
		documentAnswers,
		documentsPerWord,
		leadingWord,
		joinedDocuments,
		documentsWithoutMetadata,
		urlMetadataAnswers,
		time.Since(startedAt),
	))

	return findingsFrom(
		query,
		documentAnswers,
		documentsPerWord,
		joinedDocuments,
		urlMetadataAnswers,
	)
}

func withinHalfOfTheTimeLeft(ctx context.Context) (context.Context, context.CancelFunc) {
	deadline, bounded := ctx.Deadline()
	if !bounded {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, time.Until(deadline)/2)
}
