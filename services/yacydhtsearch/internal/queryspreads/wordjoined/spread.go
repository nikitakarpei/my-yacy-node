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
	documentsAsker           documentasks.Asker
	queryWordDocumentAmounts RememberedDocumentAmounts
	leadingWordFinder        leadingword.Finder
	urlMetadataAsker         urlmetadataasks.Asker
	observer                 WordJoinedSpreadObserver
}

func New(
	documentsAsker documentasks.Asker,
	queryWordDocumentAmounts RememberedDocumentAmounts,
	leadingWordFinder leadingword.Finder,
	urlMetadataAsker urlmetadataasks.Asker,
	observer WordJoinedSpreadObserver,
) Spread {
	return Spread{
		documentsAsker:           documentsAsker,
		queryWordDocumentAmounts: queryWordDocumentAmounts,
		leadingWordFinder:        leadingWordFinder,
		urlMetadataAsker:         urlMetadataAsker,
		observer:                 observer,
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

	lead := spread.leadingWordFinder.FindFor(ctx, query, inquiry)
	if chosenLead, chosen := lead.Get(); !chosen {
		inquiry.WhichDocumentsHave(query.HashesOfWordsAndCompoundWords())
	} else {
		roles := wordroles.Around(chosenLead, query)
		inquiry.WhichDocumentsHave(roles.LeadAndItsCompoundWords)
		inquiry.WhichDocumentsHavingTheseAlsoHave(
			roles.LeadAndItsCompoundWords,
			roles.OtherWords,
			chosenLead.AmountOfDocumentsInAPartition,
		)
	}
	documentAnswers := inquiry.Finish()

	documentsPerWord := documentsperword.From(query, documentAnswers)
	spread.queryWordDocumentAmounts.Remember(ctx, documentsPerWord.AmountInAPartitionPerQueryWord())
	joinedDocuments := documentsPerWord.WithEveryWord()
	documentsWithoutMetadata := documentAnswers.DocumentsWithoutMetadataAmong(joinedDocuments)
	holdersOfDocumentsWithoutMetadata := documentAnswers.HoldersOf(documentsWithoutMetadata)
	urlMetadataAnswers := spread.urlMetadataAsker.AskFor(ctx, holdersOfDocumentsWithoutMetadata)

	spread.observer.WordJoinedSpreadPerformed(ctx, performedWordJoinedSpreadFrom(
		documentAnswers,
		documentsPerWord,
		lead,
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
